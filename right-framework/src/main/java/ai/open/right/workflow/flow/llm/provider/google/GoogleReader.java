package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.provider.ProviderReader;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;
import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.core.TreeNode;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;

import java.nio.ByteBuffer;

@Slf4j
@Setter
@Getter
public class GoogleReader extends ProviderReader<GoogleRequest> {

    protected static final String EMPTY = "[]";

    protected Boolean skipFirst = true;

    public GoogleReader(ProviderReaderConfig<GoogleRequest> providerReaderConfig) throws Exception {
        super(providerReaderConfig);
    }

    @Override
    protected void buildStream() throws Exception {
        // 切换到读取模式
        this.byteBuffer.flip();
        if (this.byteBuffer.hasRemaining()) {
            ByteBuffer slice = this.prepare().slice(this.byteBuffer.position(), this.byteBuffer.remaining());
            int offset = slice.arrayOffset() + slice.position();
            int length = slice.remaining();
            if (log.isDebugEnabled()) {
                log.debug("The response chunk={} from slice", new String(slice.array(), offset, length));
            }
            try (JsonParser parser = JsonUtils.FACTORY.createParser(slice.array(), offset, length)) {
                parser.setCodec(JsonUtils.instance());
                // 直到读取完毕（STOP）
                while (this.hasNext(parser)) {
                    try {
                        // 如果没有待处理节点则终止
                        TreeNode node = parser.readValueAsTree();
                        if (node != null) {
                            String message = node.toString();
                            if (log.isDebugEnabled()) {
                                log.debug("The response message={}", message);
                            }
                            this.check(message);
                            this.completed(message);
                            this.position(parser);
                        }
                    } catch (JsonParseException e) {
                        // 防止误解析
                        if (log.isDebugEnabled()) {
                            log.debug(e.getMessage(), e);
                        }
                        break;
                    }
                }
            }
        }
        // 切换到写模式
        this.byteBuffer.compact();
    }

    protected Boolean hasNext(JsonParser parser) throws Exception {
        try {
            return parser.nextToken() != null;
        } catch (JsonParseException e) {
            if (log.isDebugEnabled()) {
                log.debug(e.getMessage(), e);
            }
            return true;
        }
    }

    protected void position(JsonParser parser) throws Exception {
        int position = (int) parser.getCurrentLocation().getByteOffset() + 1;
        this.byteBuffer.position(Math.min(position, this.byteBuffer.limit()));
    }

    @Override
    // 消息处理，推送队列
    protected void completed(String message) throws Exception {
        if (JsonUtils.like(message)) {
            super.completed(message);
        }
    }

    protected void check(String message) throws Exception {
        if (JsonUtils.array(message) && StringUtils.equalsIgnoreCase(StringUtils.trim(message), GoogleReader.EMPTY)) {
            throw new WorkflowException("The response message is invalid", ProtocolCode.C914);
        }
    }

    protected ByteBuffer prepare() throws Exception {
        if (this.request.getStream() && this.skipFirst) {
            this.byteBuffer.position(this.byteBuffer.position() + 1);
            this.skipFirst = false;
        }
        return this.byteBuffer;
    }
}