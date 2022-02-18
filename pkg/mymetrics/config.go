package mymetrics

import "github.com/prometheus/client_golang/prometheus"

var MyReconcileTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "my_reconcile_total",
	Namespace: "我们自己写的reconcile触发统计",
}, []string{"controller"})
