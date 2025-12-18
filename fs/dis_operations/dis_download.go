package dis_operations

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"sync/atomic"

	"github.com/rclone/rclone/cmd"
	"github.com/rclone/rclone/fs/operations"
	"github.com/rclone/rclone/reedsolomon"
	"github.com/spf13/cobra"
)

var downloadedCount int64

var copyCommandDefinitionForDown = &cobra.Command{
	Use: "copy source:path dest:path",
	Annotations: map[string]string{
		"groups": "Copy,Filter,Listing,Important",
	},
	Run: func(command *cobra.Command, args []string) {
		cmd.CheckArgs(2, 2, command, args)
		fsrc, srcFileName, fdst := cmd.NewFsSrcFileDst(args)
		cmd.RunWithSustainOS(true, true, command, func() error {
			if srcFileName == "" {
				fmt.Printf("%s is a directory or doesn't exist\n", args[0])
				return nil
			}
			return operations.CopyFile(context.Background(), fdst, fsrc, srcFileName, srcFileName)
		}, true)
	},
}

func Dis_Download(args []string, reSignal bool) (err error) {
	mode := ""

	if len(args) >= 3 {
		mode = args[2] // 세 번째 파라미터 (예: "optimize")
	}

	originalFileName := filepath.Base(args[0])
	_, err = GetFileInfoStruct(originalFileName)
	if err != nil {
		return err
	}
	fileInfo, err := GetFileInfoStruct(originalFileName)
	
	if err != nil {
		return err
	}
	// ① datamap.json 정보
	dataShards := fileInfo.Shard
	remoteShardCount := fileInfo.RemoteShardCount

	var distributedFileInfos []DistributedFile

	if reSignal {
		//Get Distribution list(Check 읽어서 false인 것만 들고 오기)
		distributedFileInfos, err = GetUncompletedFileInfo(originalFileName)
		if err != nil {
			return err
		}

	} else {
		//state 변경
		err = UpdateFileFlag(originalFileName, "download")
		if err != nil {
			return err
		}
		distributedFileInfos, err = GetDistributedFileStruct(originalFileName)
		if err != nil {
			return err
		}
		RRplan := makeRRPlan(dataShards)

		selected := selectShardsByPlan(distributedFileInfos, RRplan)
		distributedFileInfos = selected
		fmt.Println("[RR DOWNLOAD PLAN]")
		for remote, cnt := range RRplan {
			fmt.Printf("  %s: %d shards\n", remote, cnt)
		}

	}
	if mode == "optimize" {
		fmt.Println("Optimization mode: Pre-planned optimal download enabled")
		// ② loadbalancer.json 정보
		jsonFilePath := getLoadBalancerJsonFilePath()
		lbInfo, err := readJSON(jsonFilePath)
		if err != nil {
			return err
		}

		// 평균 다운로드 속도
		avgDown := make(map[string]float64)
		for name, info := range lbInfo.RemoteInfos {
			parts := strings.Split(name, "|")
			base := parts[0]
			avgDown[base] = info.AvgDownThroughput
		}

		// ③ remoteShardCount 이름 정규화
		normalizedOwned := make(map[string]int)
		for key, val := range remoteShardCount {
			parts := strings.Split(key, "|")
			base := parts[0]
			normalizedOwned[base] = val
		}

		// ④ 최적 분배 계산
		optimalPlan := findOptimalDownloadPlan(avgDown, normalizedOwned, dataShards, 16.0)
		fmt.Printf("Optimal Download Plan: %v\n", optimalPlan)

		// ⑤ plan에 맞는 shard만 선택
		filtered := selectShardsByPlan(distributedFileInfos, optimalPlan)
		fmt.Printf("Applying optimized selection: %d shards retained out of %d\n",
			len(filtered), len(distributedFileInfos))

		distributedFileInfos = filtered

	}

	start := time.Now()
	if err := startDownloadFileGoroutine_Worker(distributedFileInfos, originalFileName, 8); err != nil {
		return err
	}

	elapsed := time.Since(start)
	fmt.Println("Current Time:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("Time taken for dis_download: %s\n", elapsed)

	absolutePath, err := getAbsolutePath(args[1])
	if err != nil {
		return err
	}

	// Move downloaded file to destination
	// fileInfo, err := GetFileInfoStruct(originalFileName)
	// if err != nil {
	// 	return err
	// }

	checksums := make(map[string]string)
	for _, each := range distributedFileInfos {
		checksums[each.DistributedFile] = each.Checksum
	}
	
	err = reedsolomon.DoDecode(originalFileName, absolutePath, fileInfo.Padding, checksums, fileInfo.Shard, fileInfo.Parity, tryGetPassword())
	if err != nil {
		result := ShowDescription_RemoveFile(originalFileName, err)
		if result {
			err = Dis_rm([]string{originalFileName}, false)
			if err != nil {
				return err
			}
		}
		return nil
	}
	
	// change Flag and Check to false
	err = ResetCheckFlag(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("File successfully downloaded to %s\n", absolutePath)

	var distributedFiles []string
	for _, info := range distributedFileInfos {
		distributedFiles = append(distributedFiles, info.DistributedFile)
	}

	reedsolomon.DeleteShardWithFileNames(distributedFiles)
	fmt.Printf("Time taken for dis_download: %s\n", elapsed)
	return nil
}

type DownloadPlan map[string]int

func makeRRPlan(dataShards int) DownloadPlan {
	RRplan := make(DownloadPlan)

	// hard-coded remote names
	remotes := []string{
		"gdrive",
		"onedrive",
		"mega",
	}

	numRemotes := 3

	// ceil(dataShards / 3)
	perRemote := (dataShards + numRemotes - 1) / numRemotes

	total := 0
	for _, remote := range remotes {
		RRplan[remote] = perRemote
		total += perRemote
	}

	if total < dataShards {
		panic(fmt.Sprintf(
			"[RR DOWNLOAD ERROR] total shards %d < dataShards %d",
			total, dataShards,
		))
	}

	return RRplan
}


// findOptimalDownloadPlan - dataShards 개를 각 remote에 어떻게 나눌지 결정
func findOptimalDownloadPlan(remotes map[string]float64, owned map[string]int, dataShards int, shardSizeMB float64) DownloadPlan {
	bestPlan := make(DownloadPlan)
	const eps = 1e-6
	low := 0.0
	high := 1e9
	var bestTime float64

	keys := make([]string, 0, len(remotes))
	for k := range remotes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for high-low > eps {
		mid := (low + high) / 2
		sum := 0
		tempPlan := make(DownloadPlan)

		for _, k := range keys {
			capacity := int((mid * remotes[k]) / (shardSizeMB * 8))
			if capacity > owned[k] {
				capacity = owned[k]
			}
			if capacity < 0 {
				capacity = 0
			}
			tempPlan[k] = capacity
			sum += capacity
		}

		if sum >= dataShards {
			bestTime = mid
			bestPlan = tempPlan
			high = mid
		} else {
			low = mid
		}
	}

	finalPlan := make(DownloadPlan)
	remain := dataShards
	type kv struct {
		key string
		val float64
	}
	var sorted []kv
	for k, v := range remotes {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].val > sorted[j].val })

	for _, kv := range sorted {
		if remain <= 0 {
			break
		}
		alloc := bestPlan[kv.key]
		if alloc > remain {
			alloc = remain
		}
		finalPlan[kv.key] = alloc
		remain -= alloc
	}

	fmt.Printf("최소 예상 다운로드 시간: %.3fs\n", bestTime)
	return finalPlan
}

func selectShardsByPlan(files []DistributedFile, plan DownloadPlan) []DistributedFile {
	result := []DistributedFile{}
	count := make(map[string]int)
	for _, f := range files {
		// 🔧 Remote.Name 에는 대부분 "gdrive|drive" 형태로 저장됨
		parts := strings.Split(f.Remote.Name, "|")
		rname := parts[0] // "gdrive"만 사용

		if plan[rname] > count[rname] {
			result = append(result, f)
			count[rname]++
		}
	}
	return result
}

func startDownloadFileGoroutine_Worker(distributedFileInfos []DistributedFile, originalFileName string, workerCount int) (err error) {
	shardDir, err := reedsolomon.GetShardDir()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	jobs := make(chan DistributedFile, len(distributedFileInfos))

	// Worker function
	downloader := func() {
		for fileInfo := range jobs {
			if err := downloadFile(fileInfo, shardDir, originalFileName, &mu, &errs); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
			wg.Done()
		}
	}

	// Start worker goroutines
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			downloader()
			wg.Done()
		}()
	}

	// Send jobs to workers
	for _, fileInfo := range distributedFileInfos {
		wg.Add(1)
		jobs <- fileInfo
	}

	close(jobs) // Close channel to signal workers
	wg.Wait()   // Wait for all workers to finish
	fmt.Printf(
		"[DOWNLOAD RESULT] completed shards: %d / requested shards: %d\n",
		atomic.LoadInt64(&downloadedCount),
		len(distributedFileInfos),
	)

	return nil
}

func downloadFile(fileInfo DistributedFile, shardDir, originalFileName string, mu *sync.Mutex, errs *[]error) error {
	startTime := time.Now()

	hashedFileName, err := CalculateHash(fileInfo.DistributedFile)
	if err != nil {
		mu.Lock()
		*errs = append(*errs, fmt.Errorf("CalculateHash for %s: %w", fileInfo.DistributedFile, err))
		mu.Unlock()
		return err
	}

	source := fmt.Sprintf("%s:%s/%s", fileInfo.Remote.Name, remoteDirectory, hashedFileName)
	fmt.Printf("Downloading shard %s to %s\n", source, shardDir)
	downloadedFilePath := path.Join(shardDir, hashedFileName)

	if err := remoteCallCopyforDown([]string{source, shardDir}); err != nil {
		mu.Lock()
		*errs = append(*errs, fmt.Errorf("remoteCallCopyforDown for %s: %w", fileInfo.DistributedFile, err))
		mu.Unlock()
		return err
	}

	elapsedTime := time.Since(startTime)
	downloadedFile, err := os.Stat(downloadedFilePath)
	if err != nil {
		mu.Lock()
		*errs = append(*errs, fmt.Errorf("downloaded file %s does not exist", downloadedFilePath))
		mu.Unlock()
		return err
	}

	// Calculate throughput
	throughput := float64(downloadedFile.Size()) / elapsedTime.Seconds()
	throughputMbps := throughput * 8 / 1e6

	if err := ConvertFileNameForDo(hashedFileName, fileInfo.DistributedFile); err != nil {
		return fmt.Errorf("ConvertFileNameForDo for %s: %w", fileInfo.DistributedFile, err)
	}

	// Update remote info
	err = updateRemoteInfo_Down(originalFileName, fileInfo, throughputMbps, mu)
	if err != nil {
		return err
	}
	atomic.AddInt64(&downloadedCount, 1)
	return nil
}

func updateRemoteInfo_Down(originalFileName string, shardInfo DistributedFile, throughputMbps float64, mu *sync.Mutex) error {
	mu.Lock()
	err := UpdateDistributedFile_CheckFlag(originalFileName, shardInfo.DistributedFile, true)
	if err != nil {
		mu.Unlock()
		return fmt.Errorf("UpdateDistributedFileCheckFlag error: %v", err)
	}
	err = UpdateRemoteInfo(shardInfo.Remote, func(b *RemoteInfo) {
		b.UpdateThroughput(throughputMbps, 1)
	})
	mu.Unlock()
	if err != nil {
		return err
	}
	return nil
}

func getAbsolutePath(arg string) (string, error) {
	// Check if the path is absolute
	if filepath.IsAbs(arg) {
		// Return the cleaned absolute path
		return filepath.Clean(arg), nil
	}

	// If it's not absolute, resolve relative to the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %v", err)
	}

	// Join and clean the path to get the absolute version
	destinationPath := filepath.Join(cwd, arg)
	return filepath.Clean(destinationPath), nil
}

func remoteCallCopyforDown(args []string) (err error) {
	fmt.Printf("Calling remoteCallCopy with args: %v\n", args)

	// Fetch the copy command
	copyCommand := *copyCommandDefinitionForDown
	copyCommand.SetArgs(args)

	err = copyCommand.Execute()
	if err != nil {
		return fmt.Errorf("error executing copyCommand: %w", err)
	}

	return nil
}
