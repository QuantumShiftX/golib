package etcdc

import (
	"github.com/jinzhu/copier"
	configurator "github.com/zeromicro/go-zero/core/configcenter"
	"github.com/zeromicro/go-zero/core/configcenter/subscriber"
	"github.com/zeromicro/go-zero/core/logx"
	"strings"
)

type Etcd[T any] struct {
	configurator configurator.Configurator[T]
}

// NewEtcd 实例化etcd
func NewEtcd[T any](c Config) *Etcd[T] {
	var cc subscriber.EtcdConf
	_ = copier.Copy(&cc, &c)
	cc.Hosts = strings.Split(c.Host, ",")

	return &Etcd[T]{
		configurator: configurator.MustNewConfigCenter[T](configurator.Config{
			Type: "json",
		}, subscriber.MustNewEtcdSubscriber(cc)),
	}
}

func (ctr *Etcd[T]) GetConfig() (T, error) {
	var result T
	var err error

	result, err = ctr.configurator.GetConfig()
	if err != nil {
		logx.Errorf("Failed to get config: %v", err)
		return result, err
	}

	logx.Infof("Successfully retrieved config: %+v", result)
	return result, err

}

func (ctr *Etcd[T]) Listener(listener func(ec *Etcd[T])) {
	logx.Info("Adding listener for config changes")
	listener(ctr)
	ctr.configurator.AddListener(func() {
		logx.Info("Config change detected, notifying listener")
		listener(ctr)
	})
}
