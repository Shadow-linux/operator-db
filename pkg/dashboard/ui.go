package dashboard

import (
	"context"
	"github.com/gin-gonic/gin"
	v1 "github.com/shenyisyn/dbcore/pkg/apis/dbconfig/v1"
	"io/ioutil"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	metricclient "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func isChildDeploy(cfg *v1.DbConfig, objectRef corev1.ObjectReference, c client.Client) bool {
	deploy := &appv1.Deployment{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name: "dbcore-" + cfg.Name, Namespace: cfg.Namespace,
	}, deploy)
	if err != nil {
		return false
	}
	if objectRef.UID == deploy.UID {
		return true
	}
	return false
}

type AdminUi struct {
	r *gin.Engine
	c client.Client
	config *rest.Config
}

func NewAdminUi(c client.Client, config *rest.Config) *AdminUi  {

	r := gin.New()
	r.Use(errorHandler())
	r.GET("/", func(context *gin.Context) {
		context.JSON(200, gin.H{"message": "ok"})
	})

	return &AdminUi{r: r, c: c, config: config}
}

func (this *AdminUi) tops ()  {
	mc, err := metricclient.NewForConfig(this.config)
	Error(err)
	this.r.GET("/tops", func(c *gin.Context) {
		list, err := mc.MetricsV1beta1().NodeMetricses().List(c, metav1.ListOptions{})
		Error(err)
		c.JSON(200, list.Items)
	})
}

func (this *AdminUi) usage ()  {
	mc, err := metricclient.NewForConfig(this.config)
	Error(err)
	this.r.GET("/top-usage", func(c *gin.Context) {
		nodeName := c.Query("name")
		nm, err := mc.MetricsV1beta1().NodeMetricses().Get(c, nodeName, metav1.GetOptions{})
		Error(err)
		c.JSON(200, gin.H{"usage": nm.Usage})
	})
}

func (this *AdminUi) events()   {
	this.r.GET("/events/:ns/:name", func(c *gin.Context) {
		var ns, name =c.Param("ns"),c.Param("name")
		cfg := &v1.DbConfig{}
		Error(this.c.Get(c, types.NamespacedName{
			Namespace: ns, Name: name,
		}, cfg))

		eList := &corev1.EventList{}
		Error(this.c.List(c, eList, &client.ListOptions{}))
		retEvents := []corev1.Event{}
		for _, e := range eList.Items {
			//这是匹配 自定义资源 对应的 event
			//&& e.InvolvedObject.UID==cfg.UID
			if e.InvolvedObject.UID == cfg.UID && e.InvolvedObject.Name == cfg.Name {
				retEvents = append(retEvents, e)
				continue
			}
			//代表判断，当前资源是否是dbconfig 创建出来的 deployment
			if isChildDeploy(cfg, e.InvolvedObject, this.c) {
				retEvents = append(retEvents, e)
			}

		}
		c.JSON(200, retEvents)
	})
}

// 创建资源
func (this *AdminUi) postResource ()  {
	this.r.POST("/resource", func(c *gin.Context) {
		b, err := ioutil.ReadAll(c.Request.Body)
		Error(err)
		cfg := &v1.DbConfig{}
		Error(yaml.Unmarshal(b, cfg))
		if cfg.Namespace == "" {
			cfg.Namespace = "default"
		}
		Error(this.c.Create(c, cfg))
		c.JSON(200, gin.H{"message": "success"})
	})
}

// 删除资源
func (this *AdminUi) deleteResource ()  {
	this.r.DELETE("/resource/:ns/:name", func(c *gin.Context) {
		cfg := &v1.DbConfig{}
		Error(this.c.Get(c, types.NamespacedName{
			Name: c.Param("name"),
			Namespace: c.Param("ns"),
		}, cfg))
		Error(this.c.Delete(c, cfg))
		c.JSON(200, gin.H{"message": "OK"})
	})
}

func (this *AdminUi) total ()  {
	this.r.GET("/top-total", func(c *gin.Context) {
		nodeName := c.Query("name")
		nodeObj := &corev1.Node{}
		err := this.c.Get(c, types.NamespacedName{
			Name: nodeName,
		}, nodeObj)
		Error(err)
		c.JSON(200, gin.H{
			"cpu": float64(nodeObj.Status.Capacity.Cpu().MilliValue()),
			"mem": nodeObj.Status.Capacity.Name(corev1.ResourceMemory, resource.DecimalSI),
		})
	})
}

func (this *AdminUi) Start(ctx context.Context) error  {
	this.r.GET("/configs", func(c *gin.Context) {
		list := &v1.DbConfigList{}
		Error(this.c.List(ctx, list, &client.ListOptions{}))
		c.JSON(200, list.Items)
	})
	this.tops()
	this.usage()
	this.total()
	this.postResource()
	this.deleteResource()
	this.events()
	return this.r.Run(":9003")
}