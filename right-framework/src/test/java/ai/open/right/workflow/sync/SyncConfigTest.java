package ai.open.right.workflow.sync;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.CollectionUtils;

import java.util.Collections;
import java.util.HashMap;

public class SyncConfigTest {

    @Test
    public void testProvider() {
        SyncConfig config = SyncConfig.builder()
                .provider("PROVIDER")
                .build();
        Assert.assertTrue(!CollectionUtils.isEmpty(config.getMetadata()));
        config = SyncConfig.builder().provider("PROVIDER").build();
        Assert.assertFalse(CollectionUtils.isEmpty(config.getMetadata()));
        Assert.assertEquals("PROVIDER", config.getProvider());
        Assert.assertEquals("PROVIDER", config.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
    }

    @Test
    public void interval_builderDefault_equalsConstant() {
        SyncConfig config = SyncConfig.builder().build();
        Assert.assertEquals(SyncConfig.INTERVAL, config.getInterval());
        Assert.assertEquals(Integer.valueOf(5000), config.getInterval());
    }

    @Test
    public void interval_setter_getter() {
        SyncConfig config = SyncConfig.builder().build();
        config.setInterval(100);
        Assert.assertEquals(Integer.valueOf(100), config.getInterval());
        config.setInterval(2000);
        Assert.assertEquals(Integer.valueOf(2000), config.getInterval());
    }

    @Test
    public void interval_builderExplicit() {
        SyncConfig config = SyncConfig.builder().interval(3000).build();
        Assert.assertEquals(Integer.valueOf(3000), config.getInterval());
    }

    @Test
    public void workTask_setter_replacesAfterBuild() {
        WorkflowTask w1 = ObjectBuilder.buildWorkflowTask();
        WorkflowTask w2 = ObjectBuilder.buildWorkflowTask();
        SyncConfig config = SyncConfig.builder().workTask(w1).build();
        config.setWorkTask(w2);
        Assert.assertSame(w2, config.getWorkTask());
    }

    @Test
    public void timeout_setter_updatesAfterBuild() {
        SyncConfig config = SyncConfig.builder().timeout(100).build();
        config.setTimeout(999);
        Assert.assertEquals(Integer.valueOf(999), config.getTimeout());
    }

    @Test
    public void provider_setter_afterBuild_getMetadataInjectsProviderKey() {
        SyncConfig config = SyncConfig.builder().build();
        Assert.assertNull(config.getMetadata());
        config.setProvider("P2");
        Assert.assertEquals("P2", config.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        config.setProvider("P3");
        Assert.assertEquals("P3", config.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
    }

    @Test
    public void metadata_setter_thenGetMetadataWithProvider_preservesExistingEntries() {
        HashMap<String, Object> map = new HashMap<>();
        map.put("K", "V");
        SyncConfig config = SyncConfig.builder().metadata(map).provider("PX").build();
        config.setMetadata(new HashMap<>(Collections.singletonMap("K2", "V2")));
        config.setProvider("PY");
        Assert.assertEquals("V2", config.getMetadata().get("K2"));
        Assert.assertEquals("PY", config.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
    }

    @Test
    public void workflow_takeover_notifier_reQuery_setters_roundTrip() {
        SyncConfig c = SyncConfig.builder().build();
        c.setWorkflow("WF");
        c.setBiz("BZ");
        c.setTakeover("TO");
        c.setNotifier("NO");
        c.setReQuery("RQ");
        Assert.assertEquals("WF", c.getWorkflow());
        Assert.assertEquals("BZ", c.getBiz());
        Assert.assertEquals("TO", c.getTakeover());
        Assert.assertEquals("NO", c.getNotifier());
        Assert.assertEquals("RQ", c.getReQuery());
    }

    @Test
    public void biz_builderAndSetter_roundTrip() {
        SyncConfig config = SyncConfig.builder().biz("B1").build();
        Assert.assertEquals("B1", config.getBiz());
        config.setBiz("B2");
        Assert.assertEquals("B2", config.getBiz());
    }
}
