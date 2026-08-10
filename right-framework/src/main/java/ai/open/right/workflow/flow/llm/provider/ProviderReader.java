package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.WorkflowException;
import ai.open.right.listener.EventListenerService;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.SplitUtils;
import com.fasterxml.jackson.core.JacksonException;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.http.HttpEntity;
import org.apache.http.HttpResponse;
import org.apache.http.concurrent.FutureCallback;
import org.apache.http.entity.ContentType;
import org.apache.http.nio.ContentDecoder;
import org.apache.http.nio.IOControl;
import org.apache.http.nio.protocol.AbstractAsyncResponseConsumer;
import org.apache.http.protocol.HttpContext;
import org.slf4j.MDC;
import org.springframework.util.Assert;

import java.io.IOException;
import java.nio.ByteBuffer;
import java.nio.charset.MalformedInputException;
import java.nio.charset.StandardCharsets;
import java.util.Collections;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.Executor;

@Slf4j
@Getter
@Setter
public class ProviderReader<T extends ProviderRequest> extends AbstractAsyncResponseConsumer<Void> {

    public static final Map<String, Object> EXTENSION = Collections.unmodifiableMap(new HashMap<String, Object>());

    public static final String DONE = "data: [DONE]";

    protected final ProviderReaderCallback providerReaderCallback;

    protected final EventListenerService eventListenerService;

    protected final BlockingQueue<Object> messageQueue;

    protected final T request;

    protected StringBuffer expMessage;

    protected ByteBuffer byteBuffer;

    protected Integer messageCnt = 0;

    protected Integer chunkIndex = 0;

    protected Integer capacity;

    // 默认200
    protected Integer code = ProtocolCode.C200;

    public ProviderReader(ProviderReaderConfig<T> providerReaderConfig) throws Exception {
        // Buffer可能调整为图片类型Buffer，取最大值
        this.capacity = Math.max(providerReaderConfig.getBuffer(), providerReaderConfig.getCapacity());
        this.eventListenerService = providerReaderConfig.getEventListenerService();
        this.byteBuffer = ByteBuffer.allocate(providerReaderConfig.getBuffer());
        this.messageQueue = this.blockingQueue(providerReaderConfig);
        this.request = providerReaderConfig.getRequest();
        // 有先后顺序
        this.providerReaderCallback = this.buildReaderCallback(providerReaderConfig);
    }

    // 在构建时调用
    protected ProviderReaderCallback buildReaderCallback(ProviderReaderConfig<T> providerReaderConfig) throws Exception {
        return new ProviderReaderCallback((ProviderReaderConfig<ProviderRequest>) providerReaderConfig, this.messageQueue, this.request, this.request.getMessage());
    }

    protected BlockingQueue<Object> blockingQueue(ProviderReaderConfig<T> providerReaderConfig) throws Exception {
        return new ArrayBlockingQueue<Object>(providerReaderConfig.getQueue());
    }

    protected void buildStream() throws Exception {
        // 切换到读取模式
        this.byteBuffer.flip();
        if (this.byteBuffer.hasRemaining()) {
            try {
                int close = 0;
                while ((close = this.indexOf(this.byteBuffer)) > 0) {
                    this.completed(StandardCharsets.UTF_8.decode(this.byteBuffer.slice(this.byteBuffer.position(), close - this.byteBuffer.position())).toString());
                    // 需要包含/n/n
                    this.byteBuffer.position(close + 2);
                }
            } catch (MalformedInputException | JacksonException e) {
                if (log.isDebugEnabled()) {
                    log.debug(e.getMessage(), e);
                }
            }
        }
        // 切换到写模式
        this.byteBuffer.compact();
    }

    // 子类可覆盖
    protected String buildMessage(String message) throws Exception {
        return message;
    }

    // 消息处理，推送队列
    protected void completed(String message) throws Exception {
        if (this.isSuccess()) {
            this.request.appendResponse(message);
            if (!this.messageQueue.offer(this.buildMessage(message))) {
                throw new WorkflowException("The request failed to add message to queue for message=" + message);
            } else {
                this.messageCnt++;
            }
        } else {
            // 异常时
            this.expMessage = this.expMessage != null ? this.expMessage : new StringBuffer();
            this.expMessage.append(message);
        }
    }

    // 动态扩容缓存
    protected void capacity(int capacity) throws Exception {
        if (capacity <= this.byteBuffer.capacity()) {
            return;
        }
        if (log.isInfoEnabled()) {
            log.info("The request's buffer will be increased to {}", capacity);
        }
        ByteBuffer newByteBuffer = ByteBuffer.allocate(capacity);
        this.byteBuffer.flip();
        newByteBuffer.put(this.byteBuffer);
        this.byteBuffer = newByteBuffer;
    }

    @Override
    protected void onContentReceived(ContentDecoder decoder, IOControl ioControl) throws IOException {
        if (log.isDebugEnabled()) {
            log.debug("The request took {} milliseconds to receive the chunk={}", this.request.getMessage().getConsuming(), this.chunkIndex++);
        }
        try {
            do {
                // 扩检查容量再读取，防止Buffer上次未处理后的堆积
                if (this.byteBuffer.remaining() == 0) {
                    // 没到上限时，继续扩容
                    // 到了上限且buffer还满时，明确失败
                    int curr = this.byteBuffer.capacity();
                    int next = Math.min(this.capacity, curr * 2);
                    Assert.isTrue(next > curr, "The request's buffer is too large: " + curr);
                    this.capacity(next);
                }
            } while (this.hasRemain(decoder));
            if (this.request.getStream()) {
                this.buildStream();
            }
        } catch (Exception e) {
            this.notifyFailed(ioControl, e);
        } finally {
            Thread.yield();
        }
    }

    @Override
    protected void onEntityEnclosed(HttpEntity entity, ContentType contentType) {
        // 不需要特殊处理
    }

    @Override
    protected void onResponseReceived(HttpResponse httpResponse) throws IOException {
        this.prepareMDC();
        this.code = httpResponse.getStatusLine().getStatusCode();
        // 状态码检查,INFO 日志
        if (log.isInfoEnabled()) {
            log.info("The request took {} milliseconds to receive the response={}", this.request.getMessage().getConsuming(), this.code);
        }
    }

    protected Boolean hasRemain(ContentDecoder decoder) throws Exception {
        return decoder.read(this.byteBuffer) > 0;
    }

    @Override
    protected Void buildResult(HttpContext httpContext) {
        try {
            // INFO 日志
            if (log.isInfoEnabled()) {
                log.info("The request took {} milliseconds to build the result", this.request.getMessage().getConsuming());
            }
            // 处理剩余
            this.byteBuffer.flip();
            if (this.byteBuffer.hasRemaining()) {
                this.completed(StandardCharsets.UTF_8.decode(this.byteBuffer).toString());
            }
            // 如果是非2xx消息，以checkValidStatus异常优先
            this.checkValidStatus();
            // 检查是否为空消息
            this.checkValidMessage();
        } catch (Exception e) {
            this.notifyFailed(e);
        } finally {
            try {
                this.releaseMessageQueue();
                this.event();
            } catch (Exception e) {
                // 不要推送Failed
                WorkflowException.dolog(e);
            }
            Thread.yield();
        }
        return null;
    }

    protected void checkValidStatus() throws Exception {
        if (!this.isSuccess()) {
            String message = this.buildException();
            if (log.isInfoEnabled()) {
                log.info("The http code={} on {}, if `autodump` is enabled, requests will be saved automatically.", this.code, SplitUtils.join(this.request.getMessage()));
            }
            throw new WorkflowException(message, this.code);
        }
    }

    protected void releaseMessageQueue() {
        if (log.isInfoEnabled()) {
            log.info("The request {}@{} callback message={}", this.request.getMessage().getBiz(), this.request.getMessage().getWorkflow(), this.messageCnt);
        }
        // 发送关闭信号失败（直接使用Queue）
        if (!this.messageQueue.offer(ProviderReaderCallback.CLOSED)) {
            if (log.isInfoEnabled()) {
                log.info("The request send closed the message={}", ProviderReaderCallback.CLOSED);
            }
            this.released();
        }
    }

    protected String buildException() throws Exception {
        // 追加错误
        this.request.getProviderData().setResponse(this.expMessage != null ? this.expMessage.toString() : "");
        return ProtocolCode.C401.equals(this.code) ? "Permission denied by the model provider (" + this.request.getUrl() + "), please verify (code=" + this.code + ")" : System.lineSeparator() + "The internal error occurred, status code=" + this.code + ", if the request is terminated, please try again later." + System.lineSeparator();
    }

    protected void checkValidMessage() throws Exception {
        // 至少一条非空消息
        Assert.isTrue(this.messageCnt >= 1, "The request took more than one message: " + this.code);
    }

    protected Boolean isSuccess() throws Exception {
        return this.code != null && ProtocolCode.range2xx(this.code);
    }

    @Override
    protected void releaseResources() {
        if (log.isInfoEnabled()) {
            log.info("The request took {} milliseconds to release the resource", this.request.getMessage().getConsuming());
        }
    }

    // 不能抛出异常
    protected void notifyFailed(IOControl ioControl, Exception e) {
        this.notifyFailed(e);
        try {
            ioControl.shutdown();
        } catch (Exception close) {
            // 严重错误
            log.error(close.getMessage(), close);
        }
    }

    protected void notifyFailed(Exception e) {
        this.providerReaderCallback.failed(e);
    }

    protected void prepareMDC() {
        // 重新覆盖MDC
        MDC.put("trace", this.request.getMessage().getTrace());
        MDC.put("dimension", this.request.getMessage().getDimension());
    }

    // 开始监听队列
    public FutureCallback<Void> consuming(Executor executor) throws Exception {
        executor.execute(this.providerReaderCallback);
        if (log.isDebugEnabled()) {
            log.debug("The request started consuming messages");
        }
        return this.providerReaderCallback;
    }

    protected int indexOf(ByteBuffer buffer) throws Exception {
        int start = buffer.position();
        int end = buffer.limit();
        if (end - start < 2) {
            return -1;
        }
        int i = start;
        // 每次处理8字节
        for (; i <= end - 8; i += 8) {
            long word = buffer.getLong(i);
            // 检测long中是否包含 0x0A (\n)
            if (hasByte(word, (byte) 10) != 0) {
                // 细化搜索：一旦发现 \n\n 立即返回，确保是首次
                // 里搜索范围要稍微多扩1位，处理\n在边界的情况
                for (int j = i; j < i + 8 && j < end - 1; j++) {
                    if (buffer.get(j) == 10 && buffer.get(j + 1) == 10) {
                        return j;
                    }
                }
            }
            // 边界检查补充，如果当前long的最后一个字节是\n，而下一个long的第一个字节也是\n
            if (buffer.get(i + 7) == 10 && i + 8 < end && buffer.get(i + 8) == 10) {
                return i + 7;
            }
        }
        // 处理剩余不足8字节的部分（或者因为接近limit无法使用getLong部分）
        for (; i < end - 1; i++) {
            if (buffer.get(i) == 10 && buffer.get(i + 1) == 10) {
                return i;
            }
        }
        return -1;
    }

    // 检查long中是否存在目标字节
    protected long hasByte(long x, byte b) throws Exception {
        long mask = (b & 0xFFL) * 0x0101010101010101L;
        long val = x ^ mask;
        return (val - 0x0101010101010101L) & ~val & 0x8080808080808080L;
    }

    // 终止监听队列
    public void released() {
        if (log.isInfoEnabled()) {
            log.info("The request released");
        }
        this.providerReaderCallback.released();
    }

    // 记录事件
    protected void event() {
        try {
            // Response缓存转为String
            this.request.getProviderData().init();
            if (this.eventListenerService != null) {
                this.eventListenerService.listen(new ProviderEvent(this.request));
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }
}
