package kazoo

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	// By default, assume we're using Sarama's vagrant cluster when running tests
	zookeeperPeers []string = []string{"192.168.100.67:2181", "192.168.100.67:2182", "192.168.100.67:2183", "192.168.100.67:2184", "192.168.100.67:2185"}
)

func init() {
	if zookeeperPeersEnv := os.Getenv("ZOOKEEPER_PEERS"); zookeeperPeersEnv != "" {
		zookeeperPeers = strings.Split(zookeeperPeersEnv, ",")
	}

	fmt.Printf("Using Zookeeper cluster at %v\n", zookeeperPeers)
}

func TestBrokers(t *testing.T) {
	kz, err := NewKazoo(zookeeperPeers, nil)
	if err != nil {
		t.Fatal(err)
	}

	brokers, err := kz.Brokers()
	if err != nil {
		t.Fatal(err)
	}

	if len(brokers) == 0 {
		t.Error("Expected at least one broker")
	}

	for id, addr := range brokers {
		if conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond); err != nil {
			t.Errorf("Failed to connect to Kafka broker %d at %s", id, addr)
		} else {
			_ = conn.Close()
		}
	}

	assertSuccessfulClose(t, kz)
}

func TestTopics(t *testing.T) {
	kz, err := NewKazoo(zookeeperPeers, nil)
	if err != nil {
		t.Fatal(err)
	}

	topics, err := kz.Topics()
	if err != nil {
		t.Error(err)
	}

	if topic, ok := topics["consumergroup.multi"]; !ok {
		t.Error("Expected topic consumergroup.multi to be returned")
	} else if topic.Name != "consumergroup.multi" {
		t.Error("Expected topic consumergroup.multi to have its name set")
	}

	if _, ok := topics["__nonexistent__"]; ok {
		t.Error("Expected __nonexistent__ topic to not be defined")
	}

	assertSuccessfulClose(t, kz)
}

func TestTopicPartitions(t *testing.T) {
	kz, err := NewKazoo(zookeeperPeers, nil)
	if err != nil {
		t.Fatal(err)
	}

	partitions, err := kz.Topic("consumergroup.multi").Partitions()
	if err != nil {
		t.Fatal(err)
	}

	if len(partitions) != 4 {
		t.Errorf("Expected consumergroup.multi to have 4 partitions")
	}

	brokers, err := kz.Brokers()
	if err != nil {
		t.Fatal(err)
	}

	for partitionID, partition := range partitions {
		if partition.ID != partitionID {
			t.Error("partition.ID is not set properly")
		}

		leader, err := partition.Leader()
		if err != nil {
			t.Fatal(err)
		}

		if _, ok := brokers[leader]; !ok {
			t.Errorf("Expected the leader of consumergroup.multi/%d to be an existing broker.", partitionID)
		}

		isr, err := partition.ISR()
		if err != nil {
			t.Fatal(err)
		}

		for _, brokerID := range isr {
			if _, ok := brokers[brokerID]; !ok {
				t.Errorf("Expected all ISRs of consumergroup.multi/%d to be existing brokers.", partitionID)
			}
		}
	}

	assertSuccessfulClose(t, kz)
}

func TestTopicConfig(t *testing.T) {
	kz, err := NewKazoo(zookeeperPeers, nil)
	if err != nil {
		t.Fatal(err)
	}

	topicConfig, err := kz.Topic("consumergroup.multi").Config()
	if err != nil {
		t.Error(err)
	}
	if topicConfig["retention.ms"] != "604800000" {
		t.Error("Expected retention.ms config for consumergroup.multi to be set to 604800000")
	}

	topicConfig, err = kz.Topic("consumergroup.single").Config()
	if err != nil {
		t.Error(err)
	}
	if len(topicConfig) > 0 {
		t.Error("Expected no topic level configuration to be set for consumergroup.single")
	}

	assertSuccessfulClose(t, kz)
}

func assertSuccessfulClose(t *testing.T, kz *Kazoo) {
	if err := kz.Close(); err != nil {
		t.Error(err)
	}
}
