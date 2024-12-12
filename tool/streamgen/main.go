package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		s := <-c
		log.Fatalf("streamgen: captured signal %s\n", s)
	}()

	inputs := []string{
		`{"@level":"info","@message":"Terraform 0.15.4","@module":"terraform.ui","@timestamp":"2021-05-25T13:32:41.275359-04:00","terraform":"0.15.4","type":"version","ui":"0.1.0"}`,
		`{"@level":"info","@message":"random_pet.dog: Plan to create","@module":"terraform.ui","@timestamp":"2021-05-25T13:32:41.705503-04:00","change":{"resource":{"addr":"random_pet.dog","module":"","resource":"random_pet.dog","implied_provider":"random","resource_type":"random_pet","resource_name":"dog","resource_key":null},"action":"create"},"type":"planned_change"}`,
		`{"@level":"info","@message":"random_pet.cat: Plan to create","@module":"terraform.ui","@timestamp":"2021-05-25T13:32:41.705503-04:00","change":{"resource":{"addr":"random_pet.cat","module":"","resource":"random_pet.cat","implied_provider":"random","resource_type":"random_pet","resource_name":"cat","resource_key":null},"action":"create"},"type":"planned_change"}`,
		`{"@level":"info","@message":"random_pet.mouse: Plan to create","@module":"terraform.ui","@timestamp":"2021-05-25T13:32:41.705503-04:00","change":{"resource":{"addr":"random_pet.mouse","module":"","resource":"random_pet.mouse","implied_provider":"random","resource_type":"random_pet","resource_name":"mouse","resource_key":null},"action":"create"},"type":"planned_change"}`,
		`{"@level":"info","@message":"Plan: 3 to add, 0 to change, 0 to destroy.","@module":"terraform.ui","@timestamp":"2021-05-25T13:32:41.705638-04:00","changes":{"add":3,"change":0,"remove":0,"operation":"plan"},"type":"change_summary"}`,
		`{"@level":"info","@message":"random_pet.dog: Creating...","@module":"terraform.ui","@timestamp":"2021-05-25T13:32:41.825308-04:00","hook":{"resource":{"addr":"random_pet.dog","module":"","resource":"random_pet.dog","implied_provider":"random","resource_type":"random_pet","resource_name":"dog","resource_key":null},"action":"create"},"type":"apply_start"}`,
		`{"@level":"info","@message":"random_pet.cat: Creating...","@module":"terraform.ui","@timestamp":"2021-05-25T13:32:41.825308-04:00","hook":{"resource":{"addr":"random_pet.cat","module":"","resource":"random_pet.cat","implied_provider":"random","resource_type":"random_pet","resource_name":"cat","resource_key":null},"action":"create"},"type":"apply_start"}`,
		`{"@level":"info","@message":"random_pet.mouse: Creating...","@module":"terraform.ui","@timestamp":"2021-05-25T13:32:41.825308-04:00","hook":{"resource":{"addr":"random_pet.mouse","module":"","resource":"random_pet.mouse","implied_provider":"random","resource_type":"random_pet","resource_name":"mouse","resource_key":null},"action":"create"},"type":"apply_start"}`,
		`{"@level":"info","@message":"random_pet.dog: Creation complete after 0s [id=smart-lizard]","@module":"terraform.ui","@timestamp":"2021-05-25T13:32:41.826179-04:00","hook":{"resource":{"addr":"random_pet.dog","module":"","resource":"random_pet.dog","implied_provider":"random","resource_type":"random_pet","resource_name":"dog","resource_key":null},"action":"create","id_key":"id","id_value":"smart-lizard","elapsed_seconds":0},"type":"apply_complete"}`,
		`{"@level":"info","@message":"random_pet.cat: Creation complete after 0s [id=smart-lizard]","@module":"terraform.ui","@timestamp":"2021-05-25T13:32:41.826179-04:00","hook":{"resource":{"addr":"random_pet.cat","module":"","resource":"random_pet.cat","implied_provider":"random","resource_type":"random_pet","resource_name":"cat","resource_key":null},"action":"create","id_key":"id","id_value":"smart-lizard","elapsed_seconds":0},"type":"apply_complete"}`,
		`{"@level":"info","@message":"random_pet.mouse: Creation complete after 0s [id=smart-lizard]","@module":"terraform.ui","@timestamp":"2021-05-25T13:32:41.826179-04:00","hook":{"resource":{"addr":"random_pet.mouse","module":"","resource":"random_pet.mouse","implied_provider":"random","resource_type":"random_pet","resource_name":"mouse","resource_key":null},"action":"create","id_key":"id","id_value":"smart-lizard","elapsed_seconds":0},"type":"apply_complete"}`,
		`{"@level":"info","@message":"Apply complete! Resources: 3 added, 0 changed, 0 destroyed.","@module":"terraform.ui","@timestamp":"2021-05-25T13:32:41.869168-04:00","changes":{"add":3,"change":0,"remove":0,"operation":"apply"},"type":"change_summary"}`,
		`{"@level":"info","@message":"Outputs: 1","@module":"terraform.ui","@timestamp":"2021-05-25T13:32:41.869280-04:00","outputs":{"pets":{"sensitive":false,"type":"string","value":"smart-lizard"}},"type":"outputs"}`,
	}

	for _, input := range inputs {
		fmt.Println(input)
		time.Sleep(time.Second * 1)
	}
}