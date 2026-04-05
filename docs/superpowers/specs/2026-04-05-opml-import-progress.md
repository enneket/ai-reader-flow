# OPML 导入进度设计

## 目标

OPML 导入时显示实时进度，类似现有的刷新进度轮询机制。

## 当前状态

OPML 导入是同步的，循环调用 `AddFeed`，完成后才返回结果。用户只能看到"导入中..."，不知道具体进度。

## 设计方案

### Backend 改动

**1. 新增 Import Job 状态存储**

在 `cmd/server/main.go` 中添加内存 map 存储导入进度：

```go
type importJob struct {
    Total      int
    Current    int
    FeedName   string
    Success    int
    Failed     int
    Done       bool
    CreatedAt  time.Time
}

var importJobs = make(map[string]*importJob)
var importJobsMu sync.Mutex
var importOperationMu sync.Mutex // 防止并发导入
```

**2. 修改 `handleImportOPML` 为异步**

- 解析 OPML 获取 URL 列表
- 生成 job ID，立即返回 202 + job ID
- goroutine 循环添加 feed，每添加一个更新进度
- 完成后删除 job（1小时后自动清理）

```go
func handleImportOPML(w http.ResponseWriter, r *http.Request) {
    // ...验证...

    if !importOperationMu.TryLock() {
        http.Error(w, "another import in progress", http.StatusConflict)
        return
    }

    urls, err := opml.Import(r.Body)
    if err != nil {
        importOperationMu.Unlock()
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if len(urls) == 0 {
        importOperationMu.Unlock()
        writeJSON(w, http.StatusOK, map[string]any{"imported": 0, "message": "no feeds found"})
        return
    }

    jobID := fmt.Sprintf("%d", time.Now().UnixNano())
    importJobsMu.Lock()
    importJobs[jobID] = &importJob{
        Total:     len(urls),
        CreatedAt: time.Now(),
    }
    importJobsMu.Unlock()

    go func() {
        defer importOperationMu.Unlock()

        for _, url := range urls {
            feed, err := rssService.AddFeed(url)
            importJobsMu.Lock()
            job := importJobs[jobID]
            job.Current++
            if err != nil {
                job.Failed++
                job.FeedName = url // 显示失败的 URL
            } else {
                job.Success++
                if feed != nil {
                    job.FeedName = feed.Title
                }
            }
            importJobsMu.Unlock()
        }

        importJobsMu.Lock()
        job := importJobs[jobID]
        job.Done = true
        importJobsMu.Unlock()

        // 1小时后清理
        go func() {
            time.Sleep(time.Hour)
            importJobsMu.Lock()
            delete(importJobs, jobID)
            importJobsMu.Unlock()
        }()
    }()

    writeJSON(w, http.StatusAccepted, map[string]any{"jobId": jobID})
}
```

**3. 新增 `GET /api/opml/import/{jobId}`**

```json
{
  "current": 3,
  "total": 10,
  "feedName": "Hacker News",
  "success": 2,
  "failed": 1,
  "done": false
}
```

### API 设计

**POST /api/opml/import**
- 返回: `{"jobId": "xxx"}` (202 Accepted)
- 如有导入进行中: 409 Conflict

**GET /api/opml/import/{jobId}**
- 返回进度 JSON
- job 不存在: 404

### 前端轮询

- 调用导入后，每 200ms 轮询进度（与刷新进度一致）
- done=true 时停止轮询，显示成功/失败数

### 错误处理

- 某个 feed 添加失败: 计入 failed，继续下一个
- OPML 解析失败: 400 立即返回
- 并发导入: 返回 409 Conflict

### 文件改动

- `cmd/server/main.go`: 添加 importJobs map 和 importOperationMu，修改 handleImportOPML，新增 GET handler
- `frontend/src/components/Settings.tsx`: 添加轮询逻辑和进度显示
- `frontend/src/api.ts`: 添加 `getImportProgress(jobId)` API

### 前端显示

```
正在导入 3/10: Hacker News
[████████░░░░░░░░░░] 30%
成功: 2, 失败: 1
```
