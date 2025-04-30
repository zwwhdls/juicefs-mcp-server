package juicefs

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (j *JuiceFSHandler) handleDocsOfReadAmplification(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	j.log.Debugw("handleDocsOfReadAmplification")
	return mcp.NewToolResultText("读放大现象是：对象存储的下行流量，远大于实际读文件的速度。通过分析 accesslog，若发现读文件的行为是频繁随机小读。尤其 offset（也就是 read 的第三个参数）跳跃巨大，说明相邻的读操作之间跨度很大，难以利用到预读提前下载下来的数据。建议将 --prefetch 调整为 0，从而禁用预读行为。"), nil
}

func (j *JuiceFSHandler) handleDocsOfWriteAmplification(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	j.log.Debugw("handleDocsOfWriteAmplification")
	return mcp.NewToolResultText(`高频小追加写、随机写带来的写放大是不可避免的，这是 JuiceFS 为了读写性能所做的取舍。但是在低速顺序写场景下的碎片合并问题，我们可以用下方步骤进行甄别和优化：
	1. 进行大文件顺序写入时，期望不产生任何碎片，打开文件系统的监控页面，查看对象存储流量面板
	2. 如果发现碎片合并的流量太大，但是则可能是遇到了写入慢的碎片问题
	3. 定位到负责写入的 JuiceFS 客户端，调整挂载参数 --flush-wait=60，将默认 5 秒一次的持久化改为 60 秒，能够减少碎片量。
	`), nil
}

func (j *JuiceFSHandler) handleDocsOfMetaCache(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	j.log.Debugw("handleDocsOfMetaCache")
	return mcp.NewToolResultText(`
可以通过 FUSE 挂载参数来设置内核元数据缓存时间：
--attrcacheto=1
--entrycacheto=1
--direntrycacheto=1
entry 缓存是随着文件访问逐渐建立起来的，不是一个完整列表，因此不能被 readdir 调用或者 ls 命令使用，而只对 lookup 调用有加速效果。

以上元数据缓存项同时还存在于客户端内存中，相当于组成了「内核 → 客户端内存」的多级缓存。考虑到客户端内存中的元数据缓存会默认保留 5 分钟，并且支持主动失效，不建议将上述内核元数据的缓存时间进一步提高。如果要提高内核元数据缓存时间，文件系统应该满足以下特点：
1. 文件极少变动，或者完全只读
2. 需要 lookup 大量文件，比如对于超大型 Git 仓库运行 git status，希望尽可能避免请求穿透到用户态，获得极致性能
3. 在实际场景中，也很少需要对 --entrycacheto 和 --direntrycacheto 进行区分设置，如果确实要精细化调优，在目录极少变动、而文件频繁变动的场景，可以令 --direntrycacheto 大于 --entrycacheto。

为了减少客户端和元数据服务之间频繁的列表（ls）和查询操作，JuiceFS 默认会把访问过的文件和目录完整地缓存在客户端内存中。与内核元数据缓存不同，这部分缓存数据支持主动失效（从元数据服务获取数据变更信息，异步地清理客户端内存中的元数据），因此默认设置更长的缓存时间（5 分钟）：
--metacache # 在客户端中缓存元数据，默认开启
--metacacheto=300 # 内存元数据的缓存时间，单位为秒，默认 5 分钟
--max-cached-inodes=500000 # 默认最多会缓存 500000 个 inodes
	`), nil
}

func (j *JuiceFSHandler) handleDocsOfDataCache(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	j.log.Debugw("handleDocsOfDataCache")
	return mcp.NewToolResultText(`
JuiceFS 对数据提供多种缓存机制来提高性能，包括内核中的页缓存（Page Cache）和客户端所在机器的本地缓存，以及客户端自身的内存读写缓冲区。读请求会依次尝试内核分页缓存、JuiceFS 进程的预读缓冲区、本地磁盘缓存，当缓存中没找到对应数据时才会从对象存储读取，并且会异步写入各级缓存保证下一次访问的性能。

## 内核页缓存
对于已经读过的文件，内核会为其建立页缓存（Page Cache），延时可低至 10 微秒，吞吐量可以到每秒几 GiB。
页缓存受内核直接管理，可以通过 juicefs stats 命令，查看 fuse.read，为 0 说明读请求并未到达 FUSE，而是直接读取内核页缓存，该请求不再由 JuiceFS 客户端服务。

## 内核回写模式
FUSE 支持内核回写（writeback-cache）模式，内核会把高频随机小 IO（例如 10-100 字节）的写请求合并起来，显著提升随机写入的性能。但其副作用是会将顺序写变为随机写，严重降低顺序写的性能。
在挂载命令通过 -o writeback_cache 选项来开启内核回写模式。

## 客户端读缓存
JuiceFS 会默认将所有读取的数据缓存到本地。而在写的时候，则会默认把小于块大小（4M）的数据写入到缓存目录中，因为 4M 的数据块通常是顺序写大文件所产生的，在大部分场景下缓存价值不高。

以下是缓存配置的关键参数：

### --cache-size 与 --free-space-ratio
缓存空间大小（单位 MiB，默认 102400）与缓存盘的最少剩余空间占比（默认 0.1）。这两个参数任意一个达到阈值，均会自动触发缓存淘汰，淘汰算法使用 2-random 策略，在大部分实际应用场景下，这样的策略接近 LRU，并且开销更低。

### --cache-partial-only
读取数据的时候，仅缓存小于一个块大小（默认 4M）的数据块。相当于缓存不足 4M 的小文件，以及大文件末尾不足一个块大小的数据块。
当本地磁盘的吞吐反而比不上对象存储时，可以考虑启用 --cache-partial-only，连续读的对象块不会被缓存。而随机读（例如读 Parquet 或者 ORC 文件的 footer）所读取的字节数比较小，不会读取整个对象块，此类场景读取的数据块就会被缓存。充分地利用了本地磁盘低时延和网络高吞吐的优势。

### --cache-large-write
用来控制「写数据的时候，哪些数据会被缓存」。因为更多场景下，顺序写的缓存价值不大，因此 JuiceFS 客户端默认不会将大文件顺序写进行缓存，而是只缓存小于块大小的写入。如果开启 --cache-large-write，那么所有完整的数据块，都会随着写入而被缓存。

## 客户端写缓存
使用 --writeback 开启客户端写缓存。启用客户端写缓存时，写入流程为「先提交，再异步上传」，数据写入到本地缓存目录并提交到元数据服务后就立即返回，本地缓存目录中的文件数据会在后台异步上传至对象存储。
由于写缓存的使用注意事项较多，使用不当极易出问题，推荐仅在大量写入小文件时临时开启。
	`), nil
}

func (j *JuiceFSHandler) handleDocsOfBufferSize(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	j.log.Debugw("handleDocsOfBufferSize")
	return mcp.NewToolResultText(`
读写缓冲区是分配给 JuiceFS 客户端进程的一块内存，通过 --buffer-size 控制着大小，默认 300（单位 MiB）。读和写产生的数据，都会经过这个缓冲区。

## 预读与预取
预读和预取都会将下载下来的数据存放在缓冲区。

## 写入
write 会将数据写入缓冲区，fsync 或者当写入超过块大小（默认 4M），或者在缓冲区停留超过 5 秒（可以使用 --flush-wait 调整），才会触发自动持久化。
缓冲区是读写共用的，并且「写」具有更高的优先级。如果对象存储的上传速度不足以支撑写入负载，会发生缓冲区拥堵。若对象存储上传速度不足，写也可能会因为 flush 超时而最终失败。

## 读写缓冲区的调优
调整缓冲区大小前，可以使用 juicefs stats 来观察当前的缓冲区用量大小。
缓冲区对性能的影响是多方面的，除了预读窗口，还间接控制对象存储的请求并发度。即设置更大的 --max-downloads 或者 --max-uploads，并不一定会带来性能提升，还可能要提高缓冲区大小。

1. --max-uploads 可以增大 block 的上传并发度，同时需要调整 --buffer-size 来使得并发线程更容易申请到内存。
2. 如果客户端处在一个低带宽的网络环境下，可能需要降低 --buffer-size 来避免 flush 超时。
3. 希望增加顺序读速度，可以增加 --buffer-size，来放大预读窗口，同时也要增加 --max-downloads 来提升预读的并发度。
`), nil
}

func (j *JuiceFSHandler) handleDocsOfReadAheadAndPrefetch(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	j.log.Debugw("handleDocsOfReadAheadAndPrefetch")
	return mcp.NewToolResultText(`
## 预读
顺序读文件时，JuiceFS 客户端会进行预读（readahead），提前将文件后续的内容下载下来。
预读窗口大小会根据缓冲区和下载并发度进行推算，在 { buffer-size / 5, block-size * max-downloads, block-size * 128MiB } 中取最小值。
缓冲区对性能的影响是多方面的，除了预读窗口，还间接控制对象存储的请求并发度。即设置更大的 --max-downloads 或者 --max-uploads，并不一定会带来性能提升，还可能要提高缓冲区大小。

## 预取
JuiceFS 还支持预取（prefetch）：读取文件某个块（Block）的一小段时，客户端会异步将整个对象存储块下载下来。但是对于大文件的偏移极大的、稀疏的随机读，prefetch 会带来读放大，可通过 --prefetch=0 禁用该行为。
`), nil
}
