package ai.open.right.workflow.flow.llm;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.UserContext;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.notify.Notifier;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

public class LLMQueryTest {

    @Test
    public void testBuildWithWorkflow() {
        LLMQuery llm = LLMQuery.build(ObjectBuilder.buildWorkflowTask(), "WR");
        assertEquals("UNKNOWN", llm.getConversation());
        assertEquals(Integer.valueOf(1), llm.getDeepness());
        assertEquals("endpoint", llm.getNotifier());
        assertEquals("UNKNOWN", llm.getQuery());
        assertEquals("UNKNOWN", llm.getBiz());
        assertEquals("UNKNOWN", llm.getChat());
        assertEquals("WR", llm.getWorkflow());
        assertEquals("chat", llm.getProtocol());
        assertEquals("UNKNOWN", llm.getTrace());
        assertNotNull(llm.getMetadata());
        assertNotNull(llm.getCreated());
        assertNotNull(llm.getUserContext());
    }

    @Test
    public void testBuild() {
        LLMQuery llm = LLMQuery.build(ObjectBuilder.buildWorkflowTask());
        assertEquals("UNKNOWN", llm.getConversation());
        assertEquals(Integer.valueOf(1), llm.getDeepness());
        assertEquals("endpoint", llm.getNotifier());
        assertEquals("UNKNOWN", llm.getWorkflow());
        assertEquals("UNKNOWN", llm.getQuery());
        assertEquals("UNKNOWN", llm.getBiz());
        assertEquals("UNKNOWN", llm.getChat());
        assertEquals("chat", llm.getProtocol());
        assertEquals("UNKNOWN", llm.getTrace());
        assertNotNull(llm.getMetadata());
        assertNotNull(llm.getCreated());
        assertNotNull(llm.getUserContext());
    }

    @Test
    public void testBuildWithWorkflowAndNotifier() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        // 测试指定 workflow 和 notifier
        LLMQuery llm = LLMQuery.build(task, "MY_WORKFLOW", "MY_NOTIFIER");
        assertEquals("MY_WORKFLOW", llm.getWorkflow());
        assertEquals("MY_NOTIFIER", llm.getNotifier());

        // 测试 workflow 为空时使用默认值 "def"
        LLMQuery llmDefault = LLMQuery.build(task, "", "MY_NOTIFIER");
        assertEquals("def", llmDefault.getWorkflow());
    }

    @Test
    public void testLLMQueryChecker() {
        LLMQuery llm = LLMQuery.build(ObjectBuilder.buildWorkflowTask());
        // 正常情况不应抛出异常
        assertDoesNotThrow(() -> LLMQuery.LLMQueryChecker.check(llm));
    }

    @Test
    public void testLLMQueryCheckerWithInvalidQuery() {
        NettyRequest task = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        LLMQuery llm = LLMQuery.build(task);

        // 边界测试：conversation 为 null
        task.setConversation(null);
        assertThrows(IllegalArgumentException.class, () -> LLMQuery.LLMQueryChecker.check(llm));
        task.setConversation("conv");

        // 边界测试：UserContext 为 null
        task.setUserContext(null);
        assertThrows(IllegalArgumentException.class, () -> LLMQuery.LLMQueryChecker.check(llm));
        task.setUserContext(ObjectBuilder.buildEmpty());

        // 边界测试：Timestamp 为 null
        task.setCreated(null);
        assertThrows(IllegalArgumentException.class, () -> LLMQuery.LLMQueryChecker.check(llm));
        task.setCreated(System.currentTimeMillis());

        // 边界测试：Workflow 为空
        task.setWorkflow("");
        assertThrows(IllegalArgumentException.class, () -> LLMQuery.LLMQueryChecker.check(llm));
        task.setWorkflow("wf");

        // 边界测试：Notifier 为空
        task.setNotifier("");
        assertThrows(IllegalArgumentException.class, () -> LLMQuery.LLMQueryChecker.check(llm));
        task.setNotifier("notifier");

        // 边界测试：Query 为 null
        task.setQuery(null);
        assertThrows(IllegalArgumentException.class, () -> LLMQuery.LLMQueryChecker.check(llm));
        task.setQuery("query");

        // 边界测试：Chat 为 null
        task.setChat(null);
        assertThrows(IllegalArgumentException.class, () -> LLMQuery.LLMQueryChecker.check(llm));
        task.setChat("chat");

        // 边界测试：Biz 为空
        task.setBiz("");
        assertThrows(IllegalArgumentException.class, () -> LLMQuery.LLMQueryChecker.check(llm));
        task.setBiz("biz");

        // 边界测试：UserContext 内部字段校验 (例如 language 为 null)
        task.getUserContext().setLanguage(null);
        assertThrows(IllegalArgumentException.class, () -> LLMQuery.LLMQueryChecker.check(llm));
        task.getUserContext().setLanguage(UserContext.UNKNOWN);
    }

    @Test
    public void testCallMethods() {
        LLMQuery llm = LLMQuery.build(ObjectBuilder.buildWorkflowTask());

        // 测试切换到 LocalHost
        llm.callToLocalHost();
        assertEquals(Notifier.LOCALHOST, llm.getNotifier());

        // 测试切换到 Endpoint
        llm.callToEndpoint();
        assertEquals(Notifier.ENDPOINT, llm.getNotifier());
    }
}

