package measurements

import (
	"bufio"
	"context"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/kube-burner/kube-burner/pkg/config"
	"github.com/kube-burner/kube-burner/pkg/measurements/metrics"
	"github.com/kube-burner/kube-burner/pkg/measurements/types"
	"github.com/kube-burner/kube-burner/pkg/util/fileutils"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	dnsLatencyMeasurement          = "dnsLatencyMeasurement"
	dnsLatencyQuantilesMeasurement = "dnsLatencyQuantilesMeasurement"
)

type dnsLatency struct {
	BaseMeasurement
}

type dnsMetric struct {
	Timestamp  time.Time `json:"timestamp"`
	Avg        float64   `json:"avgLatency"`
	Min        float64   `json:"minLatency"`
	Max        float64   `json:"maxLatency"`
	MetricName string    `json:"metricName"`
	UUID       string    `json:"uuid"`
	Namespace  string    `json:"namespace"`
	Name       string    `json:"pod"`
	JobName    string    `json:"jobName,omitempty"`
	Metadata   any       `json:"metadata,omitempty"`
}

type dnsLatencyMeasurementFactory struct {
	BaseMeasurementFactory
}

func newDnsLatencyMeasurementFactory(configSpec config.Spec, measurement types.Measurement, metadata map[string]any) (MeasurementFactory, error) {
	return dnsLatencyMeasurementFactory{
		BaseMeasurementFactory: NewBaseMeasurementFactory(configSpec, measurement, metadata),
	}, nil
}

func (dlmf dnsLatencyMeasurementFactory) NewMeasurement(jobConfig *config.Job, clientSet kubernetes.Interface, restConfig *rest.Config, embedCfg *fileutils.EmbedConfiguration) Measurement {
	return &dnsLatency{
		BaseMeasurement: dlmf.NewBaseLatency(jobConfig, clientSet, restConfig, dnsLatencyMeasurement, dnsLatencyQuantilesMeasurement, embedCfg),
	}
}

func (d *dnsLatency) Start(measurementWg *sync.WaitGroup) error {
	defer measurementWg.Done()
	d.startMeasurement([]MeasurementWatcher{})
	return nil
}

func (d *dnsLatency) Stop() error {
	return d.StopMeasurement(d.normalizeMetrics, d.getLatency)
}

func (d *dnsLatency) Collect(measurementWg *sync.WaitGroup) {
	defer measurementWg.Done()
	pods, err := d.ClientSet.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{LabelSelector: "kube-burner-job=dns-latency"})
	if err != nil {
		log.Error(err)
		return
	}
	re := regexp.MustCompile(`Average latency:\s+([0-9.]+)\s+ms\s+\(min\s+([0-9.]+),\s+max\s+([0-9.]+)\)`)
	for _, pod := range pods.Items {
		req := d.ClientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
		stream, err := req.Stream(context.TODO())
		if err != nil {
			log.Errorf("error getting logs for %s/%s: %v", pod.Namespace, pod.Name, err)
			continue
		}
		scanner := bufio.NewScanner(stream)
		var avg, min, max float64
		for scanner.Scan() {
			line := scanner.Text()
			matches := re.FindStringSubmatch(line)
			if len(matches) == 4 {
				avg, _ = strconv.ParseFloat(matches[1], 64)
				min, _ = strconv.ParseFloat(matches[2], 64)
				max, _ = strconv.ParseFloat(matches[3], 64)
			}
		}
		stream.Close()
		d.Metrics.Store(string(pod.UID), dnsMetric{
			Timestamp:  pod.CreationTimestamp.UTC(),
			Avg:        avg,
			Min:        min,
			Max:        max,
			MetricName: dnsLatencyMeasurement,
			UUID:       d.Uuid,
			Namespace:  pod.Namespace,
			Name:       pod.Name,
			JobName:    d.JobConfig.Name,
			Metadata:   d.Metadata,
		})
	}
}

func (d *dnsLatency) normalizeMetrics() float64 {
	var latencies []float64
	d.Metrics.Range(func(key, value any) bool {
		metric := value.(dnsMetric)
		latencies = append(latencies, metric.Avg)
		d.NormLatencies = append(d.NormLatencies, metric)
		return true
	})
	if len(latencies) > 0 {
		summary := metrics.NewLatencySummary(latencies, "DNS")
		summary.UUID = d.Uuid
		summary.Timestamp = time.Now().UTC()
		summary.Metadata = d.Metadata
		summary.MetricName = dnsLatencyQuantilesMeasurement
		summary.JobName = d.JobConfig.Name
		d.LatencyQuantiles = append(d.LatencyQuantiles, summary)
	}
	return 0
}

func (d *dnsLatency) getLatency(metric any) map[string]float64 {
	m := metric.(dnsMetric)
	return map[string]float64{"DNS": m.Avg}
}
