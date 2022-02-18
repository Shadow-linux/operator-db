package builders

import (
	"bytes"
	"context"
	configv1 "github.com/shenyisyn/dbcore/pkg/apis/dbconfig/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"text/template"

	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMapBuilder struct {
	cm     *corev1.ConfigMap
	config *configv1.DbConfig
	client.Client
	DataKey string
}

func NewConfigMapBuilder(config *configv1.DbConfig, client client.Client) (*ConfigMapBuilder, error) {
	cm := &corev1.ConfigMap{}
	err := client.Get(context.Background(),
		types.NamespacedName{
		Namespace: config.Name, Name: deployName(config.Name),
	}, cm)
	// 没取到则拼接一个空的data
	if err != nil {
		cm.Name, cm.Namespace = deployName(config.Name), config.Namespace
		cm.Data = make(map[string]string)
	}

	return &ConfigMapBuilder{cm: cm, config: config, Client: client}, nil
}

// 设置连带关系
func (this *ConfigMapBuilder) setOwner() *ConfigMapBuilder  {
	this.cm.OwnerReferences = append(this.cm.OwnerReferences,
		metav1.OwnerReference{
			APIVersion: this.config.APIVersion,
			Kind: this.config.Kind,
			Name: this.config.Name,
			UID: this.config.UID,
		})
	return this
}

const configMapKey = "app.yml"

//把configmap里面的 key=app.yml的内容 取出变成md5,
func (this *ConfigMapBuilder) parseKey() *ConfigMapBuilder  {
	if appData, ok := this.cm.Data[configMapKey]; ok {
		this.DataKey = Md5(appData)
		return this
	}
	this.DataKey = ""
	return this
}

func (this *ConfigMapBuilder) apply() *ConfigMapBuilder  {
	// 解析模版并赋值
	tpl, err := template.New(configMapKey).Delims("[[", "]]").Parse(cmtpl)
	if err != nil {
		log.Println(err)
		return this
	}

	var tplBuffer bytes.Buffer
	if err = tpl.Execute(&tplBuffer, this.config.Spec); err != nil {
		log.Println(err)
		return this
	}

	this.cm.Data[configMapKey] = tplBuffer.String()
	return this
}

func (this *ConfigMapBuilder) Build(ctx context.Context) error {
	if this.cm.CreationTimestamp.IsZero() {
		this.apply().setOwner().parseKey()
		if err := this.Create(ctx, this.cm); err != nil {
			return err
		}
	} else {
		patch := client.MergeFrom(this.cm.DeepCopy())
		this.apply().parseKey()
		if err := this.Patch(ctx, this.cm, patch); err != nil {
			return err
		}
	}

	return nil
}
