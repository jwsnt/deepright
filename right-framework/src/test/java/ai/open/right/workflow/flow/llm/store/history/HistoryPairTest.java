package ai.open.right.workflow.flow.llm.store.history;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.SplitUtils;
import ai.open.right.netty.chat.distribute.NettyRequest;
import org.junit.Assert;
import org.junit.Test;

public class HistoryPairTest {

    /** 构造函数 HistoryPair(WorkflowTask, Long) 从 workTask 注入 created、chat、conversation、source */
    @Test
    public void testConstructorWithWorkflowTask() {
        NettyRequest workTask = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        workTask.setChat("chat-val");
        workTask.setConversation("conv-val");
        workTask.setWorkflow("WF");
        workTask.setBiz("BIZ");
        Long created = 100086L;
        HistoryPair pair = new HistoryPair(workTask, created);
        Assert.assertEquals(created, pair.getCreated());
        Assert.assertEquals("chat-val", pair.getChat());
        Assert.assertEquals("conv-val", pair.getConversation());
        Assert.assertEquals(SplitUtils.join("WF", "BIZ"), pair.getSource());
    }

    /** HistoryPair(History)：助手 + ANSWER → answer，并拷贝元数据与 function */
    @Test
    public void testConstructorWithHistory_assistantAnswer_funChat() {
        History h = new History();
        h.setFunction(History.FUN_CHAT);
        h.setConversation("conv-a");
        h.setReason("reason-a");
        h.setCreated(3001L);
        h.setSource("BIZ@WF");
        h.setChat("chat-a");
        h.setAssistant();
        h.setAnswer();
        h.setContent("assistant reply");

        HistoryPair pair = new HistoryPair(h);
        Assert.assertEquals(History.ROLE_ASSISTANT, pair.getRole());
        Assert.assertEquals(History.FUN_CHAT, pair.getFunction());
        Assert.assertEquals("conv-a", pair.getConversation());
        Assert.assertEquals("reason-a", pair.getReasoning());
        Assert.assertEquals(Long.valueOf(3001L), pair.getCreated());
        Assert.assertEquals("BIZ@WF", pair.getSource());
        Assert.assertEquals("chat-a", pair.getChat());
        Assert.assertEquals("assistant reply", pair.getAnswer());
        Assert.assertNull(pair.getQuery());
    }

    /** HistoryPair(History)：拷贝 model、api */
    @Test
    public void testConstructorWithHistory_copiesModelAndApi() {
        History h = new History();
        h.setUser();
        h.setQuery();
        h.setContent("q");
        h.setModel("mdl-x");
        h.setApi("api-y");
        HistoryPair pair = new HistoryPair(h);
        Assert.assertEquals("mdl-x", pair.getModel());
        Assert.assertEquals("api-y", pair.getApi());
    }

    /** buildHistories：query/answer 两条 History 均带上 HistoryPair 的 model、api */
    @Test
    public void testBuildHistories_propagatesModelAndApiToHistory() {
        HistoryPair historyPair = new HistoryPair();
        historyPair.setCreated(1L);
        historyPair.setQuery("Q");
        historyPair.setAnswer("A");
        historyPair.setModel("pair-model");
        historyPair.setApi("pair-api");
        History[] histories = historyPair.buildHistories();
        Assert.assertEquals("pair-model", histories[0].getModel());
        Assert.assertEquals("pair-api", histories[0].getApi());
        Assert.assertEquals("pair-model", histories[1].getModel());
        Assert.assertEquals("pair-api", histories[1].getApi());
    }

    /** HistoryPair(History)：用户 + QUERY → query，FUN_FUNCALL 保留 */
    @Test
    public void testConstructorWithHistory_userQuery_funFuncall() {
        History h = new History();
        h.setFunction(History.FUN_FUNCALL);
        h.setUser();
        h.setQuery();
        h.setContent("call tool");

        HistoryPair pair = new HistoryPair(h);
        Assert.assertEquals(History.ROLE_USER, pair.getRole());
        Assert.assertEquals(History.FUN_FUNCALL, pair.getFunction());
        Assert.assertEquals("call tool", pair.getQuery());
        Assert.assertNull(pair.getAnswer());
    }

    /** 类加载时生成的 JSON Schema 非空，且 @JsonPropertyDescription 进入 schema 文本 */
    @Test
    public void testStaticSchema() {
        Assert.assertNotNull(HistoryPair.SCHEMA_HISTORIES);
        Assert.assertFalse(HistoryPair.SCHEMA_HISTORIES.isEmpty());
        Assert.assertNotNull(HistoryPair.SCHEMA_HISTORY);
        Assert.assertFalse(HistoryPair.SCHEMA_HISTORY.isEmpty());
        Assert.assertTrue(HistoryPair.SCHEMA_HISTORY.contains("The function field: Chat=0, FunCall=1"));
        Assert.assertTrue(HistoryPair.SCHEMA_HISTORY.contains("The answer of the current interaction."));
        Assert.assertTrue(HistoryPair.SCHEMA_HISTORY.contains("The query of the current interaction."));
        Assert.assertTrue(HistoryPair.SCHEMA_HISTORY.contains("The role identifier: User=0, Assistant=1."));
        Assert.assertTrue(HistoryPair.SCHEMA_HISTORIES.contains("The function field: Chat=0, FunCall=1"));
    }

    /** 类加载时由 JSON Schema 生成的 Markdown 表非空，且字段说明与 schema 一致 */
    @Test
    public void testStaticSchemaBuildMarkdown() {
        Assert.assertNotNull(HistoryPair.SCHEMA_HISTORIES_MARKDOWN);
        Assert.assertFalse(HistoryPair.SCHEMA_HISTORIES_MARKDOWN.isEmpty());
        Assert.assertNotNull(HistoryPair.SCHEMA_HISTORY_MARKDOWN);
        Assert.assertFalse(HistoryPair.SCHEMA_HISTORY_MARKDOWN.isEmpty());
        Assert.assertTrue(HistoryPair.SCHEMA_HISTORY_MARKDOWN.contains("|Field|Type|Required|Description|"));
        Assert.assertTrue(HistoryPair.SCHEMA_HISTORIES_MARKDOWN.contains("|Field|Type|Required|Description|"));
        Assert.assertTrue(HistoryPair.SCHEMA_HISTORY_MARKDOWN.contains("The function field: Chat=0, FunCall=1"));
        Assert.assertTrue(HistoryPair.SCHEMA_HISTORY_MARKDOWN.contains("The answer of the current interaction."));
        Assert.assertTrue(HistoryPair.SCHEMA_HISTORY_MARKDOWN.contains("The query of the current interaction."));
        Assert.assertTrue(HistoryPair.SCHEMA_HISTORY_MARKDOWN.contains("The role identifier: User=0, Assistant=1."));
        Assert.assertTrue(HistoryPair.SCHEMA_HISTORIES_MARKDOWN.contains("The function field: Chat=0, FunCall=1"));
    }

    @Test
    public void test() {
        HistoryPair historyPair = new HistoryPair();
        historyPair.setCreated(100086L);
        historyPair.setSource("SOURCE");
        historyPair.setQuery("Q");
        historyPair.setAnswer("A");
        History[] histories = historyPair.buildHistories();
        Assert.assertEquals("SOURCE", historyPair.getSource());
        Assert.assertEquals("Q", histories[0].getContent());
        Assert.assertEquals(History.ROLE_USER, histories[0].getRole());
        Assert.assertEquals(Long.valueOf(100086L), histories[0].getCreated());
        Assert.assertEquals(History.TYPE_QUERY, histories[0].getType());
        Assert.assertEquals("A", histories[1].getContent());
        Assert.assertEquals(History.ROLE_ASSISTANT, histories[1].getRole());
        Assert.assertEquals(Long.valueOf(100086L), histories[1].getCreated());
        Assert.assertEquals(History.TYPE_ANSWER, histories[1].getType());
    }

    @Test
    public void test1() {
        HistoryPair historyPair = new HistoryPair();
        historyPair.setCreated(100086L);
        historyPair.setQuery("Q");
        History[] histories = historyPair.buildHistories();
        Assert.assertEquals("Q", histories[0].getContent());
        Assert.assertEquals(History.ROLE_USER, histories[0].getRole());
        Assert.assertEquals(Long.valueOf(100086L), histories[0].getCreated());
        Assert.assertEquals(History.TYPE_QUERY, histories[0].getType());
        Assert.assertNull(histories[1]);
    }

    @Test
    public void test2() {
        HistoryPair historyPair = new HistoryPair();
        historyPair.setCreated(100086L);
        historyPair.setAnswer("A");
        History[] histories = historyPair.buildHistories();
        Assert.assertNull(histories[0]);
        Assert.assertEquals("A", histories[1].getContent());
        Assert.assertEquals(History.ROLE_ASSISTANT, histories[1].getRole());
        Assert.assertEquals(Long.valueOf(100086L), histories[1].getCreated());
        Assert.assertEquals(History.TYPE_ANSWER, histories[1].getType());
    }

    @Test
    public void test3() {
        HistoryPair historyPair = new HistoryPair();
        historyPair.setCreated(100086L);
        historyPair.setAnswer("A");
        historyPair.setRole(History.ROLE_USER);
        History[] histories = historyPair.buildHistories();
        Assert.assertNull(histories[0]);
        Assert.assertNotNull(histories[1]);
        Assert.assertEquals(History.ROLE_USER, histories[1].getRole());
    }

    @Test
    public void test4() {
        HistoryPair historyPair = new HistoryPair();
        historyPair.setCreated(100086L);
        historyPair.setQuery("A");
        historyPair.setRole(History.ROLE_USER);
        History[] histories = historyPair.buildHistories();
        Assert.assertNull(histories[1]);
        Assert.assertNotNull(histories[0]);
        Assert.assertEquals(History.ROLE_USER, histories[0].getRole());
    }

    @Test
    public void testChatAndConversationPropagatedToHistories() {
        HistoryPair historyPair = new HistoryPair();
        historyPair.setCreated(100086L);
        historyPair.setQuery("Q");
        historyPair.setAnswer("A");
        historyPair.setChat("chat-123");
        historyPair.setConversation("conv-456");
        History[] histories = historyPair.buildHistories();
        Assert.assertNotNull(histories[0]);
        Assert.assertNotNull(histories[1]);
        Assert.assertEquals("chat-123", histories[0].getChat());
        Assert.assertEquals("conv-456", histories[0].getConversation());
        Assert.assertEquals("chat-123", histories[1].getChat());
        Assert.assertEquals("conv-456", histories[1].getConversation());
    }

    @Test
    public void testChatAndConversation_onlyQuery() {
        HistoryPair historyPair = new HistoryPair();
        historyPair.setCreated(100086L);
        historyPair.setQuery("Q");
        historyPair.setChat("chat-only");
        historyPair.setConversation("conv-only");
        History[] histories = historyPair.buildHistories();
        Assert.assertNotNull(histories[0]);
        Assert.assertNull(histories[1]);
        Assert.assertEquals("chat-only", histories[0].getChat());
        Assert.assertEquals("conv-only", histories[0].getConversation());
    }

    @Test
    public void testChatAndConversation_onlyAnswer() {
        HistoryPair historyPair = new HistoryPair();
        historyPair.setCreated(100086L);
        historyPair.setAnswer("A");
        historyPair.setChat("chat-answer");
        historyPair.setConversation("conv-answer");
        History[] histories = historyPair.buildHistories();
        Assert.assertNull(histories[0]);
        Assert.assertNotNull(histories[1]);
        Assert.assertEquals("chat-answer", histories[1].getChat());
        Assert.assertEquals("conv-answer", histories[1].getConversation());
    }
}
