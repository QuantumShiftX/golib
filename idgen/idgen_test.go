package idgen

import (
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"sync"
	"time"
)

// 主测试函数 - 启动100个并发测试
func runMultipleConcurrentTests(generator *IDGenX, count int) {
	fmt.Println("\n开始多重并发测试 (100组并发测试)...")

	start := time.Now()

	// 全局ID映射，用于检查所有测试中的ID是否唯一
	globalIDMap := make(map[int64]bool)
	globalMutex := sync.Mutex{}

	var testWg sync.WaitGroup
	testCount := 100 // 启动10个并发测试

	// 统计数据
	totalIDs := 0
	duplicatesFound := 0

	// 启动10个并发测试
	for t := 0; t < testCount; t++ {
		testWg.Add(1)
		go func(testIndex int) {
			defer testWg.Done()

			// 每个测试的本地ID映射
			localIDMap := make(map[int64]bool)
			localMutex := sync.Mutex{}

			var wg sync.WaitGroup
			for i := 0; i < count; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					// 每个goroutine生成一个ID
					id, err := generator.GenUserID()
					if err != nil {
						logx.Errorf("测试组 %d - 生成ID失败: %v", testIndex, err)
						return
					}

					// 检查本地ID唯一性
					localMutex.Lock()
					if localIDMap[id] {
						logx.Errorf("测试组 %d - 发现本地重复ID: %d", testIndex, id)
					}
					localIDMap[id] = true
					localMutex.Unlock()
				}(i)
			}

			wg.Wait()

			// 将本地生成的ID合并到全局映射
			globalMutex.Lock()
			for id := range localIDMap {
				if globalIDMap[id] {
					duplicatesFound++
					logx.Errorf("测试组 %d - 发现全局重复ID: %d", testIndex, id)
				} else {
					globalIDMap[id] = true
				}
			}
			totalIDs += len(localIDMap)
			globalMutex.Unlock()

			fmt.Printf("测试组 %d 完成，生成了 %d 个ID\n", testIndex, len(localIDMap))
		}(t)
	}

	testWg.Wait()
	duration := time.Since(start)

	fmt.Println("\n=========== 多重并发测试结果 ===========")
	fmt.Printf("100个测试组共生成了 %d 个ID，用时: %v\n", len(globalIDMap), duration)
	fmt.Printf("每秒生成ID数量: %.2f\n", float64(len(globalIDMap))/duration.Seconds())

	// 检查总体结果
	expectedTotal := count * testCount
	if duplicatesFound > 0 {
		fmt.Printf("警告: 预期生成 %d 个ID，实际唯一ID数 %d 个，有 %d 个重复\n",
			expectedTotal, len(globalIDMap), duplicatesFound)
	} else {
		fmt.Printf("所有ID均唯一，测试通过! 预期生成 %d 个ID，实际生成 %d 个ID\n",
			expectedTotal, len(globalIDMap))
	}
}
