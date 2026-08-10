package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.WorkflowException;
import org.apache.commons.lang3.reflect.MethodUtils;
import org.apache.http.nio.ContentDecoder;
import org.apache.http.nio.IOControl;
import org.apache.http.protocol.HttpContext;
import org.easymock.EasyMock;

import java.io.IOException;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;

public class ProviderUtils {

    public static ContentDecoder buildDecoder(ProviderReader providerReader, String message) throws Exception {
        return new SimpleInputDecoder(message.getBytes(StandardCharsets.UTF_8));
    }

    public static void buildResult(ProviderReader providerReader) {
        try {
            HttpContext httpContext = EasyMock.createMock(HttpContext.class);
            EasyMock.replay(httpContext);
            Method method = MethodUtils.getMatchingMethod(ProviderReader.class, "buildResult", HttpContext.class);
            method.setAccessible(true);
            method.invoke(providerReader, httpContext);
        } catch (Exception e) {
            throw WorkflowException.create(e);
        }
    }

    public static void invokeOnContentReceived(ProviderReader reader, ContentDecoder decoder, IOControl ioControl) throws IOException {
        try {
            Method method = MethodUtils.getMatchingMethod(reader.getClass(), "onContentReceived", ContentDecoder.class, IOControl.class);
            if (method == null) {
                throw new NoSuchMethodException("onContentReceived(ContentDecoder, IOControl) not found on " + reader.getClass().getName());
            }
            method.setAccessible(true);
            method.invoke(reader, decoder, ioControl);
        } catch (InvocationTargetException e) {
            Throwable cause = e.getCause();
            if (cause instanceof IOException) {
                throw (IOException) cause;
            }
            throw new IOException(cause != null ? cause : e);
        } catch (NoSuchMethodException | IllegalAccessException e) {
            throw new IOException(e);
        }
    }

    public static class SimpleInputDecoder implements ContentDecoder {

        private final byte[] source;

        private int offset = 0;

        private boolean completed = false;

        public SimpleInputDecoder(byte[] source) {
            this.source = source;
        }

        @Override
        public int read(ByteBuffer dst) throws IOException {
            if (completed) {
                return -1;
            }

            // 1. 计算剩余可读数据
            int chunk = source.length - offset;
            if (chunk <= 0) {
                completed = true;
                return -1;
            }

            // 2. 计算 dst 还能装多少
            int freeSpace = dst.remaining();
            if (freeSpace <= 0) {
                // 注意：Apache NIO 规范中，Buffer 满了通常返回 0
                return 0;
            }

            // 3. 决定本次读取长度
            int bytesToRead = Math.min(chunk, freeSpace);

            // 4. 拷贝数据
            dst.put(source, offset, bytesToRead);
            offset += bytesToRead;

            // 5. 检查是否读完
            if (offset >= source.length) {
                completed = true;
            }

            return bytesToRead;
        }

        @Override
        public boolean isCompleted() {
            return completed;
        }
    }
}
