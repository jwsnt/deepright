package ai.open.right.integration;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.sync.SyncConfig;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.CollectionUtils;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class RightConfigTest {

    @Test
    public void test() {
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        List<History> histories = new ArrayList<>();
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().funCallTrack("funCall").mediaContext(mediaContext).histories(histories).query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build();
        rightConfig.setTakeover("TK");
        Assert.assertEquals("TK", rightConfig.getTakeover());
        Assert.assertEquals(mediaContext, rightConfig.getMediaContext());
        Assert.assertEquals(histories, rightConfig.getHistories());
        Assert.assertEquals("funCall", rightConfig.getFunCallTrack());
        Assert.assertEquals("Query", rightConfig.getQuery());
        Assert.assertEquals("Biz", rightConfig.getBiz());
        Assert.assertEquals("Trace", rightConfig.getTrace());
        Assert.assertEquals("Chat", rightConfig.getChat());
        Assert.assertEquals(Integer.valueOf(10000), rightConfig.getTimeout());
        Assert.assertEquals(Integer.valueOf(10000), rightConfig.getTimeout(1000));
        Assert.assertEquals("Conversation", rightConfig.getConversation());
        Assert.assertEquals(userContext, rightConfig.getUserContext());
        Assert.assertEquals("Upstream", rightConfig.getUpstream());
        Assert.assertEquals("Notifier", rightConfig.getNotifier());
        Assert.assertEquals("Protocol", rightConfig.getProtocol());
        Assert.assertEquals(metadata, rightConfig.getMetadata());
        Assert.assertEquals("Workflow", rightConfig.getWorkflow());
    }

    @Test
    public void testNotifier() {
        RightConfig rightConfig = RightConfig.builder().build();
        Assert.assertNull(rightConfig.getNotifier());
    }

    @Test
    public void testProvider() {
        RightConfig config = RightConfig.builder().build();
        Assert.assertTrue(CollectionUtils.isEmpty(config.getMetadata()));
        config = RightConfig.builder().provider("PROVIDER").build();
        Assert.assertFalse(CollectionUtils.isEmpty(config.getMetadata()));
        Assert.assertEquals("PROVIDER", config.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
    }

    @org.junit.jupiter.api.Test
    public void testInit() {
        RightConfig config = RightConfig.builder().workflow("test-workflow").build();
        config.init();
        org.junit.jupiter.api.Assertions.assertEquals("test-workflow", config.getUpstream());
    }

    @org.junit.jupiter.api.Test
    public void testGetTimeoutDefault() {
        RightConfig config = RightConfig.builder().build();
        org.junit.jupiter.api.Assertions.assertNull(config.getTimeout());
        org.junit.jupiter.api.Assertions.assertEquals(Integer.valueOf(5000), config.getTimeout(5000));
    }

    @org.junit.jupiter.api.Test
    public void testSetGetObjectQueryComplex() throws Exception {
        RightConfig config = RightConfig.builder().build();
        Map<String, String> complexObj = new HashMap<>();
        complexObj.put("key", "value");
        config.setObjectQuery(complexObj);
        Map result = config.getObjectQuery(Map.class);
        org.junit.jupiter.api.Assertions.assertEquals("value", result.get("key"));
    }

    @org.junit.jupiter.api.Test
    public void testGetMetadataWithProviderNullMeta() {
        RightConfig config = RightConfig.builder().provider("test-provider").build();
        config.setMetadata(null);
        Map<String, Object> metadata = config.getMetadata();
        org.junit.jupiter.api.Assertions.assertNotNull(metadata);
        org.junit.jupiter.api.Assertions.assertEquals("test-provider", metadata.get(ProviderRequestService.KEY_PROVIDER));
    }

    @org.junit.jupiter.api.Test
    public void testSettersAndGetters() {
        RightConfig config = RightConfig.builder().build();
        config.setChatTrack(true);
        org.junit.jupiter.api.Assertions.assertTrue(config.getChatTrack());

        config.setNotifierWriteBack(null);
        org.junit.jupiter.api.Assertions.assertNull(config.getNotifierWriteBack());

        config.setSyncCallable(null);
        org.junit.jupiter.api.Assertions.assertNull(config.getSyncCallable());
    }

    /** markQuery 的 getter/setter，供 RightTask.resetQuery/markQuery 使用 */
    @org.junit.jupiter.api.Test
    public void testMarkQueryGetSet() {
        RightConfig config = RightConfig.builder().query("q").build();
        org.junit.jupiter.api.Assertions.assertNull(config.getMarkQuery());
        config.setMarkQuery("marked");
        org.junit.jupiter.api.Assertions.assertEquals("marked", config.getMarkQuery());
        config.setQuery("current");
        config.setMarkQuery(config.getQuery());
        org.junit.jupiter.api.Assertions.assertEquals("current", config.getMarkQuery());
    }
}

