package builders

import (
	"bytes"
	"context"
	"fmt"
	configv1 "github.com/shenyisyn/dbcore/pkg/apis/dbconfig/v1"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"text/template"
)

type DeployBuilder struct {
	client.Client
	config    *configv1.DbConfig
	cmBuilder *ConfigMapBuilder
	deploy    *v1.Deployment
}

//目前软件的 命名规则
func deployName(name string) string {
	return "dbcore-" + name
}

func NewDeployBuilder(config *configv1.DbConfig, c client.Client) (*DeployBuilder, error) {
	deploy := &v1.Deployment{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name: deployName(config.Name), Namespace: config.Namespace,
	}, deploy)
	if err != nil {

		deploy.Name, deploy.Namespace = config.Name, config.Namespace
		tpl, err := template.New("deploy").Parse(deployTpl)
		if err != nil {
			return nil, err
		}

		var doc bytes.Buffer
		err = tpl.Execute(&doc, deploy)
		if err != nil {
			return nil, err
		}

		err = yaml.Unmarshal(doc.Bytes(), deploy)
		if err != nil {
			return nil, err
		}

	}

	cmBuilder, err := NewConfigMapBuilder(config, c)
	if err != nil {
		log.Println("cm error:", err)
		return nil, err
	}

	return &DeployBuilder{
		deploy: deploy,
		Client: c, config: config,
		cmBuilder: cmBuilder}, nil

}

func (this *DeployBuilder) apply() *DeployBuilder {
	*this.deploy.Spec.Replicas = int32(this.config.Spec.Replicas)
	return this
}

const CMAnnotation = "dbcore.config/md5"

func (this *DeployBuilder) setCMAnnotation(configStr string) {
	this.deploy.Spec.Template.Annotations[CMAnnotation] = configStr
}

// 设置属主，用作资源关联，删除时可连带删除
func (this *DeployBuilder) setOwner() *DeployBuilder {
	this.deploy.OwnerReferences = append(this.deploy.OwnerReferences,
		metav1.OwnerReference{
			APIVersion: this.config.APIVersion,
			Kind:       this.config.Kind,
			Name:       this.config.Name,
			UID:        this.config.UID,
		},
	)
	return this
}

func (this *DeployBuilder) Build(ctx context.Context) error {
	// 如果没有creationTime 是nil，则创建deploy
	if this.deploy.CreationTimestamp.IsZero() {

		this.apply().setOwner()

		if err := this.cmBuilder.Build(ctx); err != nil {
			log.Printf("Build ConfigMap Error: %+v \n", err)
		}

		//设置 config md5
		this.setCMAnnotation(this.cmBuilder.DataKey)

		if err := this.Create(ctx, this.deploy); err != nil {
			log.Printf("Create Deployment Error: %+v \n", err)
			return err
		}
	} else {
		if err := this.cmBuilder.Build(ctx); err != nil {
			log.Printf("Build ConfigMap Error: %+v \n", err)
		}
		// 合并原有对象
		patch := client.MergeFrom(this.deploy.DeepCopy())
		fmt.Printf("=== Patch: %+v \n", patch)
		this.apply()
		//设置 config md5
		this.setCMAnnotation(this.cmBuilder.DataKey)
		if err := this.Patch(ctx, this.deploy, patch); err != nil {
			log.Printf("Update Deployment Error: %+v \n", err)
		}

	}
	// 查看状态
	// 获取ready replicas
	readyReplicas := this.deploy.Status.ReadyReplicas
	this.config.Status.ReadyReplicas = fmt.Sprintf("%d/%d", readyReplicas, this.config.Spec.Replicas)
	this.config.Status.Replicas = *this.deploy.Spec.Replicas
	if err := this.Status().Update(ctx, this.config); err != nil {
		return err
	}

	return nil
}
