package ai.open.right.workflow.flow.llm.rag;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import org.junit.Assert;
import org.junit.Test;

public class RagConditionTest {
    @Test
    public void testAllowedWithOutCondition() throws Exception {
        RagCondition ragCondition = new RagCondition();
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder().config(new LLMConfig()).prompt("HELLO").query(query).build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        Assert.assertTrue(ragCondition.allowed(ragConfig, ragData));
    }

    @Test
    public void testAllowedWithTrue() throws Exception {
        RagCondition ragCondition = new RagCondition();
        ragCondition.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("true"));
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder().config(new LLMConfig()).prompt("HELLO").query(query).build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        ragConfig.setReplace("#key");
        Assert.assertTrue(ragCondition.allowed(ragConfig, ragData));
    }

    @Test
    public void testAllowedWithYes() throws Exception {
        RagCondition ragCondition = new RagCondition();
        ragCondition.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("yes"));
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder().config(new LLMConfig()).prompt("HELLO").query(query).build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        ragConfig.setReplace("#key");
        Assert.assertTrue(ragCondition.allowed(ragConfig, ragData));
    }

    @Test
    public void testAllowedWithT() throws Exception {
        RagCondition ragCondition = new RagCondition();
        ragCondition.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("t"));
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder().config(new LLMConfig()).prompt("HELLO").query(query).build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        ragConfig.setReplace("#key");
        Assert.assertTrue(ragCondition.allowed(ragConfig, ragData));
    }

    @Test
    public void testAllowedWithY() throws Exception {
        RagCondition ragCondition = new RagCondition();
        ragCondition.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("y"));
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder().config(new LLMConfig()).prompt("HELLO").query(query).build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        ragConfig.setReplace("#key");
        Assert.assertTrue(ragCondition.allowed(ragConfig, ragData));
    }

    @Test
    public void testAllowedWith1() throws Exception {
        RagCondition ragCondition = new RagCondition();
        ragCondition.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("1"));
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder().config(new LLMConfig()).prompt("HELLO").query(query).build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        ragConfig.setReplace("#key");
        Assert.assertTrue(ragCondition.allowed(ragConfig, ragData));
    }

    @Test
    public void testAllowedWithFalse() throws Exception {
        RagCondition ragCondition = new RagCondition();
        ragCondition.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false"));
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder().config(new LLMConfig()).prompt("HELLO").query(query).build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        ragConfig.setReplace("#key");
        Assert.assertFalse(ragCondition.allowed(ragConfig, ragData));
    }

    @Test
    public void testAllowedWithNo() throws Exception {
        RagCondition ragCondition = new RagCondition();
        ragCondition.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("no"));
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder().config(new LLMConfig()).prompt("HELLO").query(query).build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        ragConfig.setReplace("#key");
        Assert.assertFalse(ragCondition.allowed(ragConfig, ragData));
    }

    @Test
    public void testAllowedWithF() throws Exception {
        RagCondition ragCondition = new RagCondition();
        ragCondition.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("f"));
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder().config(new LLMConfig()).prompt("HELLO").query(query).build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        ragConfig.setReplace("#key");
        Assert.assertFalse(ragCondition.allowed(ragConfig, ragData));
    }

    @Test
    public void testAllowedWithN() throws Exception {
        RagCondition ragCondition = new RagCondition();
        ragCondition.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("n"));
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder().config(new LLMConfig()).prompt("HELLO").query(query).build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        ragConfig.setReplace("#key");
        Assert.assertFalse(ragCondition.allowed(ragConfig, ragData));
    }

    @Test
    public void testAllowedWith0() throws Exception {
        RagCondition ragCondition = new RagCondition();
        ragCondition.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("0"));
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder().config(new LLMConfig()).prompt("HELLO").query(query).build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        ragConfig.setReplace("#key");
        Assert.assertFalse(ragCondition.allowed(ragConfig, ragData));
    }
    @Test(expected = IllegalArgumentException.class)
    public void testAllowedEmptyResponse() throws Exception {
        RagCondition ragCondition = new RagCondition();
        ragCondition.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(""));
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder().config(new LLMConfig()).prompt("HELLO").query(query).build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        ragCondition.allowed(ragConfig, ragData);
    }
}
