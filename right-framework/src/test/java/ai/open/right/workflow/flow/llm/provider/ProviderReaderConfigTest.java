package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.listener.EventListenerService;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

public class ProviderReaderConfigTest {

    private static ProviderReaderConfig.ProviderReaderConfigBuilder<ProviderRequest> baseBuilder() {
        return ProviderReaderConfig.<ProviderRequest>builder()
                .eventListenerService(EasyMock.createMock(EventListenerService.class))
                .notifierService(EasyMock.createMock(NotifierService.class))
                .llmCallback(EasyMock.createMock(LLMCallback.class))
                .extension(new HashMap<>())
                .discard(1)
                .timeout(1)
                .buffer(1)
                .capacity(1)
                .queue(1)
                .request(new ProviderRequest());
    }

    @Test
    public void testCheck_success_returnsThis() throws Exception {
        ProviderReaderConfig<ProviderRequest> config = baseBuilder().build();
        Assert.assertSame(config, config.check());
    }

    @Test
    public void testBuild_allFields_populatesGetters() {
        Map<String, Object> ext = new HashMap<>();
        ext.put("k", "v");
        ProviderReaderConfig<ProviderRequest> config = baseBuilder()
                .extension(Collections.unmodifiableMap(ext))
                .discard(1)
                .timeout(2)
                .buffer(3)
                .capacity(3)
                .queue(4)
                .build();
        Assert.assertNotNull(config.getEventListenerService());
        Assert.assertNotNull(config.getNotifierService());
        Assert.assertNotNull(config.getLlmCallback());
        Assert.assertEquals("v", config.getExtension().get("k"));
        Assert.assertEquals(Integer.valueOf(1), config.getDiscard());
        Assert.assertEquals(Integer.valueOf(2), config.getTimeout());
        Assert.assertEquals(Integer.valueOf(3), config.getBuffer());
        Assert.assertEquals(Integer.valueOf(3), config.getCapacity());
        Assert.assertEquals(Integer.valueOf(4), config.getQueue());
        Assert.assertNotNull(config.getRequest());
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheck_nullNotifierService() throws Exception {
        baseBuilder().notifierService(null).build().check();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheck_nullLlmCallback() throws Exception {
        baseBuilder().llmCallback(null).build().check();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheck_nullExtension() throws Exception {
        baseBuilder().extension(null).build().check();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheck_nullDiscard() throws Exception {
        baseBuilder().discard(null).build().check();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheck_nullTimeout() throws Exception {
        baseBuilder().timeout(null).build().check();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheck_nullRequest() throws Exception {
        baseBuilder().request(null).build().check();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheck_nullBuffer() throws Exception {
        baseBuilder().buffer(null).build().check();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheck_nullCapacity() throws Exception {
        baseBuilder().capacity(null).build().check();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheck_nullQueue() throws Exception {
        baseBuilder().queue(null).build().check();
    }
}
