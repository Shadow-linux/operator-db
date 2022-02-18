package k8sConfig

import (
	"fmt"
	v1 "github.com/shenyisyn/dbcore/pkg/apis/dbconfig/v1"
	"github.com/shenyisyn/dbcore/pkg/controllers"
	"github.com/shenyisyn/dbcore/pkg/dashboard"
	"github.com/shenyisyn/dbcore/pkg/mymetrics"
	appv1 "k8s.io/api/apps/v1"
	"log"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func InitManager()  {

	logf.SetLogger(zap.New())
	mgr, err := manager.New(K8sRestConfig(), manager.Options{
		Logger: logf.Log.WithName("dbcore"),
		// 开启控制器选主
		LeaderElection: true,
		LeaderElectionID: "dbcore-lock",
		LeaderElectionNamespace: "default",
		MetricsBindAddress: ":8082",
	})

	//加入自定义指标
	metrics.Registry.MustRegister(mymetrics.MyReconcileTotal)

	if err != nil {
		log.Fatal(fmt.Sprintf("unable to setup manager, err: %s", err.Error()))
	}

	if err = v1.SchemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		mgr.GetLogger().Error(err, "unable add scheme")
	}
	// 获取event recorder
	dbconfigController:=controllers.NewDbConfigController(mgr.GetEventRecorderFor("dbconfig"))
	if err = builder.ControllerManagedBy(mgr).
		For(&v1.DbConfig{}).
		Watches(&source.Kind{Type: &appv1.Deployment{}},
			handler.Funcs{
				UpdateFunc: dbconfigController.OnUpdate,
				DeleteFunc: dbconfigController.OnDelete,
			},
			).
		Complete(dbconfigController); err != nil {
			mgr.GetLogger().Error(err, "unable to create manager.")
			os.Exit(1)
	}

	// 启动管理器，只要实现Start接口皆可以通过mgr 启动
	if err = mgr.Add(dashboard.NewAdminUi(mgr.GetClient(), K8sRestConfig())); err != nil {
		mgr.GetLogger().Error(err, "unable to start dashboard")
		os.Exit(1)
	}

	if err = mgr.Start(signals.SetupSignalHandler()); err != nil {
		mgr.GetLogger().Error(err, "unable to start manager")
		os.Exit(1)
	}
}
