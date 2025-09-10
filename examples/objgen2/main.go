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
	"sync"
	"time"

	"github.com/pingcap/log"
	"github.com/tikv/client-go/v2/config"
	"github.com/tikv/client-go/v2/rawkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	var cfg zap.Config
	cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	cfg.Encoding = "json"
	cfg.OutputPaths = []string{"stderr"}
	cfg.ErrorOutputPaths = []string{"stderr"}
	cfg.InitialFields = map[string]interface{}{"component": "tikv-client"}
	cfg.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:     "time",
		MessageKey:  "message",
		LevelKey:    "level",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
		EncodeTime:  zapcore.RFC3339NanoTimeEncoder,
	}
	zapLogger, err := cfg.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to setup TiKV logger: %w", err))
	}

	// This atomically replaces the global logger which is used by the tikv
	// golang client packages.
	log.ReplaceGlobals(zapLogger, &log.ZapProperties{Level: zap.NewAtomicLevelAt(zapcore.DebugLevel)})

	cli, err := rawkv.NewClient(context.TODO(), []string{"127.0.0.1:2379"}, config.DefaultConfig().Security)
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	key := []byte("Company")

	if err = cli.Delete(context.Background(), key); err != nil {
		panic(fmt.Sprintf("failed to delete key"))
	}

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {

		wg.Add(1)
		go func(threadId int) {
			defer wg.Done()

			objCtx := &config.ObjGen2Context{
				ObjectName: "foo",
				StartTime:  time.Now(),
				ClusterId:  threadId,
				Op:         "write",
			}

			ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
			ctx = context.WithValue(ctx, config.ObjGen2Key, objCtx)
			defer cancel()

			val := []byte(fmt.Sprintf("%d", i))
			start := time.Now()
			fmt.Printf("Thread %d: putting key %s\n", threadId, string(val))
			err = cli.Put(ctx, key, val)
			diff := time.Now().Sub(start)
			if err != nil {
				fmt.Printf("Thread %d: error putting key %v diff=%v\n", threadId, err, diff)
			} else {
				fmt.Printf("Thread %d: successfully put key %s %v\n", threadId, string(val), diff)
			}
		}(i)
	}

	wg.Wait()

}
