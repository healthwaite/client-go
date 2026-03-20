// Copyright 2021 TiKV Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"

	"github.com/tikv/client-go/v2/config"
	"github.com/tikv/client-go/v2/rawkv"
)

func main() {
	cli, err := rawkv.NewClient(context.TODO(), []string{"127.0.0.1:2379"}, config.DefaultConfig().Security)
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	fmt.Printf("cluster ID: %d\n", cli.ClusterID())

	key := []byte("key")
	val := []byte("v1")

	// insert key via compare and swap
	cli.SetAtomicForCAS(true)
	_, success, err := cli.CompareAndSwap(context.TODO(), key, nil, val)
	if err != nil || !success {
		panic(fmt.Sprintf("insert key failed %v %v\n", err, success))
	}
	fmt.Printf("Successfully inserted key via compare and swap\n")

	// update key via compare and swap
	_, success, err = cli.CompareAndSwap(context.TODO(), key, val, []byte("v2"))
	if err != nil || !success {
		panic(fmt.Sprintf("update key failed %v %v\n", err, success))
	}
	fmt.Printf("Successfully update key via compare and swap\n")

	// try to update via compare and swap but with an incorrect previous value
	incorrectVal := []byte("v42")
	_, success, err = cli.CompareAndSwap(context.TODO(), key, incorrectVal, []byte("v3"))
	if err != nil {
		panic(fmt.Sprintf("update key errored %v %v\n", err, success))
	}
	if success {
		panic("expected swap to fail")
	}

	// delete the key via compare and delete
	_, success, err = cli.CompareAndDelete(context.TODO(), key, []byte("v2"))
	if err != nil || !success {
		panic(fmt.Sprintf("delete key failed %v %v\n", err, success))
	}
	fmt.Printf("Successfully deleted key via compare and delete\n")

	// try to delete the key again, it should fail
	_, success, err = cli.CompareAndDelete(context.TODO(), key, []byte("v2"))
	if err != nil {
		panic(fmt.Sprintf("update key errored %v %v\n", err, success))
	}
	if success {
		panic("expected delete to fail")
	}

	// check the key is gone via a get
	val, err = cli.Get(context.TODO(), key)
	if err != nil {
		panic(fmt.Sprintf("get key errored %v %v\n", err, success))
	}
	if val != nil {
		panic(fmt.Sprintf("expected value to be nil, instead got %s\n", string(val)))
	}
	fmt.Printf("Successfully checked key is removed via Get()\n")
}
