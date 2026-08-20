package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.notify.NotifierService;
import org.apache.http.HttpVersion;
import org.apache.http.message.BasicHttpResponse;
import org.junit.Assert;
import org.junit.Test;
import org.mockito.ArgumentCaptor;
import org.mockito.Mockito;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ArrayBlockingQueue;

public class ProviderHeaderTest {

    @Test
    public void testAppendHeaderNormalizesNamesAndKeepsLastValue() throws Exception {
        ProviderRequest request = new ProviderRequest();
        BasicHttpResponse response = new BasicHttpResponse(HttpVersion.HTTP_1_1, 200, "OK");
        response.addHeader("X-Request-Id", "first");
        response.addHeader("x-request-id", "last");
        response.addHeader("Retry-After", "30");

        this.reader(request).appendHeader(response);

        Assert.assertEquals("last", request.getProviderHeaders().get("x-request-id"));
        Assert.assertEquals("30", request.getProviderHeaders().get("retry-after"));
    }

    @Test
    public void testPutHeadersRequiresHeaderKey() throws Exception {
        ProviderRequest request = new ProviderRequest();
        request.appendHeaders(Collections.singletonMap("x-request-id", "request-1"));
        Map<String, Object> metadata = new HashMap<String, Object>();

        request.putHeaders(metadata);
        Assert.assertTrue(metadata.isEmpty());

        request.setHeaderKey("providerHeaders");
        request.putHeaders(metadata);
        Assert.assertSame(request.getProviderHeaders(), metadata.get("providerHeaders"));
    }

    @Test
    public void testHeaderKeyComesFromInternalMetadata() throws Exception {
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + "headerKey", "providerHeaders");
        LLMQuery query = ObjectBuilder.buildLLMQuery(metadata);

        Assert.assertEquals("providerHeaders", new TestProviderRequestService().headerKey(new LLMConfig(), query));
    }

    @Test
    public void testFailureNotificationContainsProviderHeaders() throws Exception {
        NotifierService notifierService = Mockito.mock(NotifierService.class);
        ProviderRequest request = new ProviderRequest();
        request.setHeaderKey("providerHeaders");
        request.appendHeaders(Collections.singletonMap("x-request-id", "request-2"));
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        ProviderReaderCallback callback = new ProviderReaderCallback(this.callbackConfig(notifierService), new ArrayBlockingQueue<Object>(1), request, workTask);

        callback.notifyException(new WorkflowException("provider failed"));

        ArgumentCaptor<Segment> segmentCaptor = ArgumentCaptor.forClass(Segment.class);
        Mockito.verify(notifierService).notify(segmentCaptor.capture(), Mockito.same(workTask));
        Map<String, Object> headers = Map.class.cast(segmentCaptor.getValue().getMetadata().get("providerHeaders"));
        Assert.assertEquals("request-2", headers.get("x-request-id"));
    }

    @Test
    public void testCompletedStreamContainsProviderHeaders() throws Exception {
        ProviderRequest request = new ProviderRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setStream(true);
        request.setHeaderKey("providerHeaders");
        request.appendHeaders(Collections.singletonMap("x-request-id", "request-3"));
        ProviderStream<ProviderRequest> stream = this.stream(request);

        stream.addProviderHeaders(true);

        Map<String, Object> headers = Map.class.cast(stream.getSegment().getMetadata().get("providerHeaders"));
        Assert.assertEquals("request-3", headers.get("x-request-id"));
    }

    private ProviderReader<ProviderRequest> reader(ProviderRequest request) throws Exception {
        return new ProviderReader<ProviderRequest>(ProviderReaderConfig.<ProviderRequest>builder()
                .request(request)
                .notifierService(Mockito.mock(NotifierService.class))
                .llmCallback(this.callback())
                .extension(new HashMap<String, Object>())
                .capacity(32)
                .discard(0)
                .timeout(1000)
                .buffer(32)
                .queue(1)
                .build());
    }

    private ProviderReaderConfig<ProviderRequest> callbackConfig(NotifierService notifierService) {
        return ProviderReaderConfig.<ProviderRequest>builder()
                .notifierService(notifierService)
                .llmCallback(this.callback())
                .discard(0)
                .timeout(1000)
                .build();
    }

    private ProviderStream<ProviderRequest> stream(ProviderRequest request) throws Exception {
        return new ProviderStream<ProviderRequest>(ProviderStreamConfig.<ProviderRequest>builder()
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return false;
            }

            @Override
            protected Boolean atonce(String source) {
                return false;
            }
        };
    }

    private LLMCallback callback() {
        return message -> {
        };
    }

    private static class TestProviderRequestService extends ProviderRequestService<ProviderRequest> {

        @Override
        protected ProviderRequest build() {
            return new ProviderRequest();
        }

        public String headerKey(LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
            return this.buildHeaderKey(new ProviderRequest(), llmConfig, llmQuery);
        }
    }
}
