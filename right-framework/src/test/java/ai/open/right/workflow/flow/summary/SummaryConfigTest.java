package ai.open.right.workflow.flow.summary;

import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

public class SummaryConfigTest {

    @Test
    public void test() {
        SummaryConfig summaryConfig = new SummaryConfig();
        summaryConfig.setRepositories(Arrays.asList("HELLO"));
        Assert.assertFalse(summaryConfig.getDesc());
        summaryConfig.setMaxsize(99);
        summaryConfig.setScene("SCENE");
        summaryConfig.setTimeout4Llm(1000);
        summaryConfig.setDesc(true);
        Assert.assertEquals(Integer.valueOf(99), summaryConfig.getMaxsize());
        Assert.assertEquals(Integer.valueOf(1000), summaryConfig.getTimeout4Llm());
        Assert.assertEquals("SCENE", summaryConfig.getScene());
        Assert.assertTrue(summaryConfig.getDesc());
        Assert.assertEquals("HELLO", summaryConfig.getRepositories().getFirst());
    }

    @Test
    public void testWithRepo1() {
        SummaryConfig summaryConfig = new SummaryConfig();
        List<String> repo = new ArrayList<String>();
        repo.add("HELLO");
        summaryConfig.setRepositories(repo);
        summaryConfig.setScene("SCENE");
        summaryConfig.setTimeout4Llm(1000);
        Assert.assertEquals(Integer.valueOf(1000), summaryConfig.getTimeout4Llm());
        Assert.assertEquals("SCENE", summaryConfig.getScene());
        Assert.assertEquals("[HELLO, SCENE]", summaryConfig.getRepositories("WORLD").toString());
    }

    @Test
    public void testWithRepo2() {
        SummaryConfig summaryConfig = new SummaryConfig();
        List<String> repo = new ArrayList<String>();
        repo.add("HELLO");
        summaryConfig.setRepositories(repo);
        summaryConfig.setTimeout4Llm(1000);
        Assert.assertEquals(Integer.valueOf(1000), summaryConfig.getTimeout4Llm());
        Assert.assertEquals("[HELLO, WORLD]", summaryConfig.getRepositories("WORLD").toString());
    }

    @Test
    public void testWithRepo3() {
        SummaryConfig summaryConfig = new SummaryConfig();
        List<String> repo = new ArrayList<String>();
        repo.add("HELLO");
        summaryConfig.setRepositories(repo);
        summaryConfig.setTimeout4Llm(1000);
        Assert.assertEquals(Integer.valueOf(1000), summaryConfig.getTimeout4Llm());
        Assert.assertEquals("[HELLO]", summaryConfig.getRepositories("HELLO").toString());
    }

    @Test
    public void testMerge() throws Exception {
        SummaryConfig base = new SummaryConfig();
        base.setRepositories(Arrays.asList("BASE_REPO"));
        base.setTimeout4Llm(500);
        base.setCondition("BASE_CONDITION");
        base.setDynamic("BASE_DYNAMIC");
        base.setExpired(3600);
        base.setScene("BASE_SCENE");
        base.setStore(false);
        SummaryConfig override = new SummaryConfig();
        override.setRepositories(Arrays.asList("OVERRIDE_REPO"));
        override.setTimeout4Llm(1000);
        override.setCondition("OVERRIDE_CONDITION");
        override.setDynamic("OVERRIDE_DYNAMIC");
        override.setExpired(7200);
        override.setScene("OVERRIDE_SCENE");
        override.setStore(true);
        override.setMaxsize(99);
        base.merge(override);
        Assert.assertEquals(Integer.valueOf(99), base.getMaxsize());
        Assert.assertEquals("BASE_REPO", base.getRepositories().getFirst());
        Assert.assertEquals("OVERRIDE_REPO", base.getRepositories().getLast());
        Assert.assertEquals(Integer.valueOf(500), base.getTimeout4Llm());
        Assert.assertEquals("BASE_CONDITION", base.getCondition());
        Assert.assertEquals("BASE_DYNAMIC", base.getDynamic());
        Assert.assertEquals(Integer.valueOf(3600), base.getExpired());
        Assert.assertEquals("BASE_SCENE", base.getScene());
        Assert.assertEquals(Boolean.FALSE, base.getStore());
    }

    @Test
    public void testMergeWithNullBaseFields() throws Exception {
        SummaryConfig base = new SummaryConfig();
        SummaryConfig override = new SummaryConfig();
        override.setRepositories(Arrays.asList("OVERRIDE_REPO"));
        override.setTimeout4Llm(1000);
        override.setCondition("OVERRIDE_CONDITION");
        override.setDynamic("OVERRIDE_DYNAMIC");
        override.setExpired(7200);
        override.setScene("OVERRIDE_SCENE");
        override.setStore(true);
        base.merge(override);
        Assert.assertEquals(Arrays.asList("OVERRIDE_REPO"), base.getRepositories());
        Assert.assertEquals(Integer.valueOf(1000), base.getTimeout4Llm());
        Assert.assertEquals("OVERRIDE_CONDITION", base.getCondition());
        Assert.assertEquals("OVERRIDE_DYNAMIC", base.getDynamic());
        Assert.assertEquals(Integer.valueOf(7200), base.getExpired());
        Assert.assertEquals("OVERRIDE_SCENE", base.getScene());
        Assert.assertEquals(Boolean.TRUE, base.getStore());
    }

    @Test
    public void testMergeWithNullOverride() throws Exception {
        SummaryConfig base = new SummaryConfig();
        base.setRepositories(Arrays.asList("BASE_REPO"));
        base.setTimeout4Llm(500);
        SummaryConfig result = base.merge(null);
        Assert.assertSame(base, result);
        Assert.assertEquals(Arrays.asList("BASE_REPO"), base.getRepositories());
        Assert.assertEquals(Integer.valueOf(500), base.getTimeout4Llm());
    }

    @Test
    public void testMergeWithPartialOverride() throws Exception {
        SummaryConfig base = new SummaryConfig();
        base.setRepositories(Arrays.asList("BASE_REPO"));
        base.setTimeout4Llm(500);
        SummaryConfig override = new SummaryConfig();
        override.setCondition("OVERRIDE_CONDITION");
        override.setStore(true);
        base.merge(override);
        Assert.assertEquals(Arrays.asList("BASE_REPO"), base.getRepositories());
        Assert.assertEquals(Integer.valueOf(500), base.getTimeout4Llm());
        Assert.assertEquals("OVERRIDE_CONDITION", base.getCondition());
        Assert.assertNull(base.getDynamic());
        Assert.assertNull(base.getExpired());
        Assert.assertNull(base.getScene());
        Assert.assertEquals(Boolean.TRUE, base.getStore());
    }

    @Test
    public void testGetSplitDefaultTrueWhenNull() {
        SummaryConfig summaryConfig = new SummaryConfig();
        Assert.assertTrue(summaryConfig.getSplit());
    }

    @Test
    public void testGetSplitWhenSetTrue() {
        SummaryConfig summaryConfig = new SummaryConfig();
        summaryConfig.setSplit(true);
        Assert.assertTrue(summaryConfig.getSplit());
    }

    @Test
    public void testGetSplitWhenSetFalse() {
        SummaryConfig summaryConfig = new SummaryConfig();
        summaryConfig.setSplit(false);
        Assert.assertFalse(summaryConfig.getSplit());
    }

    @Test
    public void testMergeSplitBaseTakesPrecedence() throws Exception {
        SummaryConfig base = new SummaryConfig();
        base.setSplit(true);
        SummaryConfig override = new SummaryConfig();
        override.setSplit(false);
        base.merge(override);
        Assert.assertTrue(base.getSplit());
    }

    @Test
    public void testMergeSplitOverrideWhenBaseNull() throws Exception {
        SummaryConfig base = new SummaryConfig();
        SummaryConfig override = new SummaryConfig();
        override.setSplit(false);
        base.merge(override);
        Assert.assertFalse(base.getSplit());
    }

    @Test
    public void testMergeSplitOverrideTrueWhenBaseNull() throws Exception {
        SummaryConfig base = new SummaryConfig();
        SummaryConfig override = new SummaryConfig();
        override.setSplit(true);
        base.merge(override);
        Assert.assertTrue(base.getSplit());
    }

    @Test
    public void testMergeSplitBaseFalsePreserved() throws Exception {
        SummaryConfig base = new SummaryConfig();
        base.setSplit(false);
        SummaryConfig override = new SummaryConfig();
        override.setSplit(true);
        base.merge(override);
        Assert.assertFalse(base.getSplit());
    }

    @Test
    public void testGetIncludeFunCallDefaultTrueWhenNull() {
        SummaryConfig config = new SummaryConfig();
        Assert.assertTrue("includeFunCall 未设置时默认 true", config.getIncludeFunCall());
    }

    @Test
    public void testGetIncludeFunCallWhenSetTrue() {
        SummaryConfig config = new SummaryConfig();
        config.setIncludeFunCall(true);
        Assert.assertTrue(config.getIncludeFunCall());
    }

    @Test
    public void testGetIncludeFunCallWhenSetFalse() {
        SummaryConfig config = new SummaryConfig();
        config.setIncludeFunCall(false);
        Assert.assertFalse(config.getIncludeFunCall());
    }

    @Test
    public void testMergeIncludeFunCallBaseTakesPrecedence() throws Exception {
        SummaryConfig base = new SummaryConfig();
        base.setIncludeFunCall(false);
        SummaryConfig override = new SummaryConfig();
        override.setIncludeFunCall(true);
        base.merge(override);
        Assert.assertFalse(base.getIncludeFunCall());
    }

    @Test
    public void testMergeIncludeFunCallOverrideWhenBaseNull() throws Exception {
        SummaryConfig base = new SummaryConfig();
        SummaryConfig override = new SummaryConfig();
        override.setIncludeFunCall(false);
        base.merge(override);
        Assert.assertFalse(base.getIncludeFunCall());
    }

    @Test
    public void testGetDropOnFailedDefaultFalseWhenNull() {
        SummaryConfig config = new SummaryConfig();
        Assert.assertFalse("dropOnFailed 未设置时默认 false", config.getDropOnFailed());
    }

    @Test
    public void testGetDropOnFailedWhenSetTrue() {
        SummaryConfig config = new SummaryConfig();
        config.setDropOnFailed(true);
        Assert.assertTrue(config.getDropOnFailed());
    }

    @Test
    public void testGetDropOnFailedWhenSetFalse() {
        SummaryConfig config = new SummaryConfig();
        config.setDropOnFailed(false);
        Assert.assertFalse(config.getDropOnFailed());
    }

    @Test
    public void testMergeDropOnFailedBaseTakesPrecedence() throws Exception {
        SummaryConfig base = new SummaryConfig();
        base.setDropOnFailed(true);
        SummaryConfig override = new SummaryConfig();
        override.setDropOnFailed(false);
        base.merge(override);
        Assert.assertTrue(base.getDropOnFailed());
    }

    @Test
    public void testMergeDropOnFailedOverrideWhenBaseNull() throws Exception {
        SummaryConfig base = new SummaryConfig();
        SummaryConfig override = new SummaryConfig();
        override.setDropOnFailed(true);
        base.merge(override);
        Assert.assertTrue(base.getDropOnFailed());
    }

    @Test
    public void testMergeDropOnFailedOverrideFalseWhenBaseNull() throws Exception {
        SummaryConfig base = new SummaryConfig();
        SummaryConfig override = new SummaryConfig();
        override.setDropOnFailed(false);
        base.merge(override);
        Assert.assertFalse(base.getDropOnFailed());
    }

    @Test
    public void testMergeDropOnFailedBaseFalsePreserved() throws Exception {
        SummaryConfig base = new SummaryConfig();
        base.setDropOnFailed(false);
        SummaryConfig override = new SummaryConfig();
        override.setDropOnFailed(true);
        base.merge(override);
        Assert.assertFalse(base.getDropOnFailed());
    }

    @Test
    public void testGetIncludeReasonDefaultTrueWhenNull() {
        SummaryConfig config = new SummaryConfig();
        Assert.assertTrue(config.getIncludeReason());
    }

    @Test
    public void testGetIncludeReasonWhenSetTrue() {
        SummaryConfig config = new SummaryConfig();
        config.setIncludeReason(true);
        Assert.assertTrue(config.getIncludeReason());
    }

    @Test
    public void testGetIncludeReasonWhenSetFalse() {
        SummaryConfig config = new SummaryConfig();
        config.setIncludeReason(false);
        Assert.assertFalse(config.getIncludeReason());
    }

    @Test
    public void testMergeIncludeReasonBaseTakesPrecedence() throws Exception {
        SummaryConfig base = new SummaryConfig();
        base.setIncludeReason(false);
        SummaryConfig override = new SummaryConfig();
        override.setIncludeReason(true);
        base.merge(override);
        Assert.assertFalse(base.getIncludeReason());
    }

    @Test
    public void testMergeIncludeReasonOverrideWhenBaseNull() throws Exception {
        SummaryConfig base = new SummaryConfig();
        SummaryConfig override = new SummaryConfig();
        override.setIncludeReason(false);
        base.merge(override);
        Assert.assertFalse(base.getIncludeReason());
    }
}
