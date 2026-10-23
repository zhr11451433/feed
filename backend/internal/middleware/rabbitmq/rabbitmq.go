package rabbitmq

import (
	"backend/internal/config"
	"errors"
	"fmt"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn *amqp.Connection
}

func NewRabbitMQ(cfg *config.RabbitMQConfig) (*RabbitMQ, error) {
	if cfg == nil || cfg.Host == "" || cfg.Port == 0 {
		return nil, errors.New("rabbitmq config is incomplete")
	}
	url := "amqp://" + cfg.Username + ":" + cfg.Password + "@" + cfg.Host + ":" + strconv.Itoa(cfg.Port) + "/"
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return &RabbitMQ{Conn: conn}, nil
}

func (r *RabbitMQ) Close() error {
	if r == nil || r.Conn == nil {
		return nil
	}
	return r.Conn.Close()
}

func (r *RabbitMQ) NewChannel() (*amqp.Channel, error) {
	if r == nil || r.Conn == nil {
		return nil, errors.New("rabbitmq connection is not initialized")
	}
	return r.Conn.Channel()
}

const DLXExchange = "dlx.events"

//声明 Topic 交换机、业务队列、绑定关系，并调用死信声明
//创建一条业务消息通道

// DeclareTopic 声明 Topic 交换机、业务队列，并建立绑定关系。
// 同时为业务队列配置死信交换机，并尝试声明对应的死信队列。
func DeclareTopic(ch *amqp.Channel, exchange, queue, bindingKey string) error {
	// 1. 检查 RabbitMQ Channel 是否已初始化。
	// 后续所有声明操作都要通过 Channel 执行。
	if ch == nil {
		return errors.New("channel is not initialized")
	}
	// 2. 检查必要参数，避免使用空名称创建 RabbitMQ 资源。
	if exchange == "" || queue == "" || bindingKey == "" {
		return errors.New("exchange/queue/bindingKey is required")
	}
	// 3. 声明 Topic 类型交换机。
	// 参数依次为：
	// 名称、类型、持久化、自动删除、内部交换机、等待响应、额外参数。
	// durable=true：交换机在 RabbitMQ 重启后可以保留。
	// autoDelete=false：不因最后一个绑定被移除而自动删除。
	if err := ch.ExchangeDeclare(
		exchange, "topic", true, false, false, false, nil,
	); err != nil {
		return err
	}
	// 4. 声明业务队列，并配置死信交换机。
	// durable=true：队列持久化。
	// autoDelete=false：消费者断开后不自动删除队列。
	// exclusive=false：不是仅当前连接独占的队列。
	// arguments 中的 x-dead-letter-exchange 指定死信交换机。
	q, err := ch.QueueDeclare(
		queue, true, false, false, false,
		amqp.Table{
			"x-dead-letter-exchange": DLXExchange,
		},
	)
	if err != nil {
		return err
	}
	// 5. 将业务队列绑定到业务交换机。
	// q.Name 是 RabbitMQ 实际返回的队列名。
	// bindingKey 是路由匹配规则，例如 "like.*"。
	// false 表示不等待 RabbitMQ 的同步响应。
	if err := ch.QueueBind(
		q.Name, bindingKey, exchange, false, nil,
	); err != nil {
		return err
	}
	// 6. 声明死信交换机、死信队列和绑定关系。
	// 这里的原始代码只记录错误，没有向上返回错误。
	if err := DeclareDLX(ch, queue); err != nil {
		return fmt.Errorf("declare DLX for queue %s: %w", queue, err)
	}
	// 7. 业务交换机、业务队列及绑定声明成功，返回 nil。
	return nil
}

//声明死信交换机、死信队列及绑定关系
//给业务队列准备一个“异常消息回收站”

// DeclareDLX 为指定业务队列声明死信交换机、死信队列和绑定规则。
func DeclareDLX(ch *amqp.Channel, queueName string) error {
	// 1. 检查 Channel。
	// 原始代码遇到 nil 时返回 nil，但这会把未执行声明误报为成功。
	if ch == nil {
		return errors.New("channel is not initialized")
	}

	// 2. 声明死信交换机，使用 Topic 类型。
	// durable=true，交换机持久化。
	if err := ch.ExchangeDeclare(
		DLXExchange, "topic", true, false, false, false, nil,
	); err != nil {
		return err
	}

	// 3. 为每个业务队列生成对应的死信队列名称。
	// 例如 like.events → like.events.dlx。
	dlxQueue := queueName + ".dlx"

	// 4. 声明死信队列。
	// durable=true，队列持久化；没有额外队列参数。
	if _, err := ch.QueueDeclare(
		dlxQueue, true, false, false, false, nil,
	); err != nil {
		return err
	}

	// 5. 将死信队列绑定到死信交换机。
	// "#" 可以匹配任意层级的路由键，包括零个路由段。
	// 因此这个队列可以接收该死信交换机上各种路由键的消息。
	return ch.QueueBind(
		dlxQueue, "#", DLXExchange, false, nil,
	)
}

//把项目里需要的所有业务队列配置集中起来，逐个创建
//项目的消息基础设施初始化入口

// DeclareTopology 集中声明项目所需的所有 RabbitMQ 消息拓扑。
func (r *RabbitMQ) DeclareTopology() error {
	// 1. 创建一个 Channel，用于执行所有声明操作。
	ch, err := r.NewChannel()
	if err != nil {
		return err
	}

	// 2. 函数结束时关闭 Channel。
	// defer 会在当前函数返回前执行。
	defer func() {
		_ = ch.Close()
	}()

	// 3. 定义项目需要创建的 Topic 交换机、业务队列和绑定规则。
	//exchange	         queue	          bindingKey
	//like.events	     like.events	     like.*
	//comment.events	comment.events	    comment.*
	//social.events	      social.events	    social.*
	//video.popularity.events	video.popularity.events	video.popularity.*
	//video.timeline.events	video.timeline.update.queue	video.timeline.*
	topics := []struct {
		exchange   string
		queue      string
		bindingKey string
	}{
		{"like.events", "like.events", "like.*"},
		{"comment.events", "comment.events", "comment.*"},
		{"social.events", "social.events", "social.*"},
		{"video.popularity.events", "video.popularity.events", "video.popularity.*"},
		{"video.timeline.events", "video.timeline.update.queue", "video.timeline.*"},
	}

	// 4. 遍历配置列表，逐个声明业务消息通道。
	for _, t := range topics {
		if err := DeclareTopic(
			ch, t.exchange, t.queue, t.bindingKey,
		); err != nil {
			return err
		}
	}

	// 5. 所有声明均成功，返回 nil。
	return nil
}
