package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/Pallinder/go-randomdata"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/pshima/consul-snapshot/backup"
	"github.com/pshima/consul-snapshot/consul"
	"github.com/pshima/consul-snapshot/restore"
)

type Seeder struct {
	Data []*consulapi.KVPair
}

type LocalDevConsul struct {
	Command     string
	CommandArgs []string
	CommandOut  []string
}

func scan(s *bufio.Scanner, c *LocalDevConsul) {
	for s.Scan() {
		c.CommandOut = append(c.CommandOut, s.Text())
	}
}

func (c *LocalDevConsul) Run() {
	cmd := exec.Command(c.Command, c.CommandArgs...)
	c.CommandOut = append(c.CommandOut, "Preparing to start...")

	reader, err := cmd.StdoutPipe()
	if err != nil {
		log.Print("Unable to create stdout pipe for command")
	}

	scanner := bufio.NewScanner(reader)

	go scan(scanner, c)

	err = cmd.Start()
	if err != nil {
		log.Printf("Unable to start: %v", err)
	}
	err = cmd.Wait()
	if err != nil {
		log.Printf("Unable to wait for command: %v", err)
	}

	err = cmd.Process.Kill()
	if err != nil {
		log.Printf("Unable to kill command: %v", err)
	}
}

func putKey(c *consulapi.Client, k string, v string, s *Seeder) error {
	writeOpt := &consulapi.WriteOptions{}
	var err error
	writeKey := &consulapi.KVPair{Key: k, Value: []byte(v)}
	s.Data = append(s.Data, writeKey)
	_, err = c.KV().Put(writeKey, writeOpt)
	if err != nil {
		return fmt.Errorf("Failed to write test key: %v", err)
	}
	return nil
}

func checkKey(c *consulapi.Client, kv *consulapi.KVPair) error {
	queryOpt := &consulapi.QueryOptions{}
	keyCheck, _, err := c.KV().Get(kv.Key, queryOpt)
	if err != nil || keyCheck == nil {
		return fmt.Errorf("Failed to get key: %v", err)
	}

	reflecttest := reflect.DeepEqual(keyCheck.Value, kv.Value)

	if reflecttest != true {
		return fmt.Errorf("Key %v did not match\n\tExpected: %v\n\tGot: %v", kv.Key, string(kv.Value), string(keyCheck.Value))
	}

	return nil
}

func TestAcceptance(t *testing.T) {
	var err error
	if os.Getenv("ACCEPTANCE_TEST") == "" {
		t.Skip("Skipping acceptance test, Set ACCEPTANCE_TEST=1 to run")
	}
	command := "consul"
	commandargs := []string{"agent", "-dev", "-bind=127.0.0.1"}
	cmd := &LocalDevConsul{Command: command, CommandArgs: commandargs}
	go cmd.Run()

	// 5-10 seconds is about the time it takes for consul to wake up
	time.Sleep(5 * time.Second)

	c := consul.ConsulClient()

	seedData := &Seeder{}

	t.Log("Adding random seed data to consul kv")
	for i := 1; i < 500; i++ {
		// more entropy is needed here.
		randname := fmt.Sprintf("%s.%s", randomdata.SillyName(), randomdata.SillyName())
		err = putKey(c, randname, randomdata.Address(), seedData)
		if err != nil {
			t.Errorf("Failed putting test data to kv: %v", err)
		}
		err = putKey(c, randomdata.IpV4Address(), randomdata.Paragraph(), seedData)
		if err != nil {
			t.Errorf("Failed putting test data to kv: %v", err)
		}
	}

	backup.BackupRunner("test")

	_, err = c.KV().DeleteTree("", nil)
	if err != nil {
		for _, i := range cmd.CommandOut {
			log.Printf("CONSUL: %v", i)
		}
		t.Errorf("Unable to clear consul kv store after backup; %v", err)
	}

	restore.RestoreRunner("test", "test")

	for _, kv := range seedData.Data {
		//log.Printf("SEED: %v | %v", kv.Key, string(kv.Value))
		err := checkKey(c, kv)
		if err != nil {
			for _, i := range cmd.CommandOut {
				log.Printf("CONSUL: %v", i)
			}
			t.Errorf("Key Failure: %v", err)
		}
	}

}
