package controllers

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	v1 "github.com/shenyisyn/dbcore/pkg/apis/dbconfig/v1"
	"github.com/shenyisyn/dbcore/pkg/builders"
	"github.com/shenyisyn/dbcore/pkg/mymetrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ResourceKind            = "DbConfig"
	ResourceApiGroupVersion = "api.jtthink.com/v1"
)

type DbConfigController struct {
	client.Client
	E record.EventRecorder // 记录事件
}

func NewDbConfigController(e record.EventRecorder) *DbConfigController {

	return &DbConfigController{E: e}
}

// 捕获更新事件
func (r *DbConfigController) OnUpdate(updateEvent event.UpdateEvent, wq workqueue.RateLimitingInterface) {
	for _, ref := range updateEvent.ObjectNew.GetOwnerReferences() {
		if ref.Kind == ResourceKind && ref.APIVersion == ResourceApiGroupVersion {
			wq.Add(reconcile.Request{
				types.NamespacedName{
					Namespace: updateEvent.ObjectNew.GetNamespace(),
					Name:      ref.Name,
				},
			})
		}
	}
}

// 捕获删除事件，重新扔会
func (r *DbConfigController) OnDelete(event event.DeleteEvent, wq workqueue.RateLimitingInterface) {

	for _, ref := range event.Object.GetOwnerReferences() {
		if ref.Kind == ResourceKind && ref.APIVersion == ResourceApiGroupVersion {
			fmt.Printf("-- ref.Name -- %s \n", ref.Name)
			fmt.Printf("-- event.Object.Name -- %s \n", event.Object.GetName())
			wq.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: event.Object.GetNamespace(),
					Name:      ref.Name,
				},
			})
		}
	}

}

func (r *DbConfigController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {

	mymetrics.MyReconcileTotal.With(prometheus.Labels{
		"controller": "dbconfig",
	}).Inc()

	config := &v1.DbConfig{}
	err := r.Get(ctx, req.NamespacedName, config)
	if err != nil {
		return reconcile.Result{}, err
	}

	bd, err := builders.NewDeployBuilder(config, r.Client)
	if err != nil {
		return reconcile.Result{}, err
	}

	// 自定义事件
	r.E.Event(config, corev1.EventTypeNormal, "初始化Deploy", "成功")

	if err = bd.Build(ctx); err != nil {
		r.E.Event(config, corev1.EventTypeWarning, "创建Deploy", err.Error())
		return reconcile.Result{}, err
	}
	r.E.Event(config, corev1.EventTypeNormal, "创建Deploy", "成功")
	fmt.Println(config)
	return reconcile.Result{}, err
}

func (r *DbConfigController) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}
