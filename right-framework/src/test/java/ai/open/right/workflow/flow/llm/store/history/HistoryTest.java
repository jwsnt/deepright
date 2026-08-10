package ai.open.right.workflow.flow.llm.store.history;

import ai.open.right.ObjectBuilder;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.Markdown;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.LLMQueryDelegate;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import com.google.common.collect.ImmutableMap;
import org.apache.commons.collections.MapUtils;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.Map;

public class HistoryTest {

    @Test
    public void test() {
        History history = new History();
        history.setContent("Content");
        history.setSource("SOURCE");
        history.setRole(History.ROLE_ASSISTANT);
        history.setType(History.TYPE_ANSWER);
        Assert.assertEquals("SOURCE", history.getSource());
        Assert.assertEquals(Integer.valueOf(1), history.getRole());
        Assert.assertEquals(Integer.valueOf(1), history.getType());
        Assert.assertEquals("Content", history.getContent());
    }

    @Test
    public void testSetRole() {
        History history = new History();
        history.setAssistant();
        Assert.assertEquals(History.ROLE_ASSISTANT, history.getRole());
        history.setUser();
        Assert.assertEquals(History.ROLE_USER, history.getRole());
        history.setAnswer();
        Assert.assertEquals(History.TYPE_ANSWER, history.getType());
        history.setQuery();
        Assert.assertEquals(History.TYPE_QUERY, history.getType());
        Assert.assertTrue(history.isRole(History.ROLE_USER));
        Assert.assertTrue(history.isType(History.TYPE_QUERY));
    }

    @Test
    public void testGetContent() throws Exception {
        History history = new History();
        history.setContent(JsonUtils.write(ImmutableMap.of("A", "B")));
        history.setRole(History.ROLE_ASSISTANT);
        history.setType(History.TYPE_ANSWER);
        Assert.assertEquals(Integer.valueOf(1), history.getRole());
        Assert.assertEquals(Integer.valueOf(1), history.getType());
        Assert.assertEquals("B", history.getObjectContent(Map.class).get("A"));
    }

    @Test
    public void testGetFunctionAsString_defaultChat() {
        History history = new History();
        Assert.assertEquals("FUN_CHAT", history.getFunctionAsString());
    }

    @Test
    public void testGetFunctionAsString_funcallRequest() {
        History history = new History();
        history.setFunction(History.FUN_FUNCALL);
        Assert.assertEquals("FUN_FUNCALL", history.getFunctionAsString());
    }

    @Test
    public void testGetFunctionAsString_unknownReturnsEmpty() {
        History history = new History();
        history.setFunction(999);
        Assert.assertEquals("", history.getFunctionAsString());
    }

    @Test
    public void testChatAndConversation() {
        History history = new History();
        history.setChat("chat-id-001");
        history.setConversation("conv-id-001");
        Assert.assertEquals("chat-id-001", history.getChat());
        Assert.assertEquals("conv-id-001", history.getConversation());
    }

    @Test
    public void testModelAndApi() {
        History history = new History();
        history.setModel("gemini-2");
        history.setApi("google");
        Assert.assertEquals("gemini-2", history.getModel());
        Assert.assertEquals("google", history.getApi());
    }

    @Test
    public void testCopy_copiesAllFieldsToNewInstance() {
        History original = new History();
        original.setConversation("conv-copy");
        original.setReference(History.REFERENCE_CLIENT);
        original.setReason("reason-copy");
        original.setFunction(History.FUN_FUNCALL);
        original.setCreated(12345L);
        original.setContent("content-copy");
        original.setSource("BIZ@COPY");
        original.setModel("model-copy");
        original.setChat("chat-copy");
        original.setType(History.TYPE_ANSWER);
        original.setRole(History.ROLE_ASSISTANT);
        original.setApi("api-copy");

        History copied = original.copy();

        Assert.assertNotSame(original, copied);
        Assert.assertEquals(original.getConversation(), copied.getConversation());
        Assert.assertEquals(original.getReference(), copied.getReference());
        Assert.assertEquals(original.getReason(), copied.getReason());
        Assert.assertEquals(original.getFunction(), copied.getFunction());
        Assert.assertEquals(original.getCreated(), copied.getCreated());
        Assert.assertEquals(original.getContent(), copied.getContent());
        Assert.assertEquals(original.getSource(), copied.getSource());
        Assert.assertEquals(original.getModel(), copied.getModel());
        Assert.assertEquals(original.getChat(), copied.getChat());
        Assert.assertEquals(original.getType(), copied.getType());
        Assert.assertEquals(original.getRole(), copied.getRole());
        Assert.assertEquals(original.getApi(), copied.getApi());
    }

    @Test
    public void testChatAndConversation_defaultNull() {
        History history = new History();
        Assert.assertNull(history.getChat());
        Assert.assertNull(history.getConversation());
    }

    @Test
    public void testStaticSchema() {
        Assert.assertNotNull(History.SCHEMA_HISTORIES);
        Assert.assertFalse(History.SCHEMA_HISTORIES.isEmpty());
        Assert.assertNotNull(History.SCHEMA_HISTORY);
        Assert.assertFalse(History.SCHEMA_HISTORY.isEmpty());
    }

    /** 静态块：SCHEMA_*_MARKDOWN 与 SCHEMA_* 同步生成且非空 */
    @Test
    public void testStaticSchemaBuildMarkdown_populated() {
        Assert.assertNotNull(History.SCHEMA_HISTORIES_MARKDOWN);
        Assert.assertFalse(History.SCHEMA_HISTORIES_MARKDOWN.isEmpty());
        Assert.assertNotNull(History.SCHEMA_HISTORY_MARKDOWN);
        Assert.assertFalse(History.SCHEMA_HISTORY_MARKDOWN.isEmpty());
        Assert.assertTrue(History.SCHEMA_HISTORIES_MARKDOWN.contains("|Field|Type|Required|Description|"));
        Assert.assertTrue(History.SCHEMA_HISTORY_MARKDOWN.contains("|Field|Type|Required|Description|"));
    }

    /** 静态块中的 markdown 与 {@link Markdown#object} 对当前 schema 的即时计算一致 */
    @Test
    public void testStaticSchemaMarkdown_matchesBuildMarkdownMdOfSchemaJson() throws Exception {
        String expectedHistories = Markdown.object(JsonUtils.write(MapUtils.getMap(JsonUtils.read(History.SCHEMA_HISTORIES, Map.class), "items")));
        Assert.assertEquals(expectedHistories, History.SCHEMA_HISTORIES_MARKDOWN);
        String expectedHistory = Markdown.object(JsonUtils.write(JsonUtils.read(History.SCHEMA_HISTORY, Map.class)));
        Assert.assertEquals(expectedHistory, History.SCHEMA_HISTORY_MARKDOWN);
    }

    @Test
    public void getReferenceHistory_nullHistories_returnsNull() {
        Assert.assertNull(History.getReferenceHistory(null, History.REFERENCE_CLIENT));
    }

    @Test
    public void getReferenceHistory_emptyHistories_returnsNull() {
        Assert.assertNull(History.getReferenceHistory(Collections.emptyList(), History.REFERENCE_CLIENT));
    }

    @Test
    public void getReferenceHistory_filtersByReference_externalOnly() {
        History internal = new History();
        internal.setReference(History.REFERENCE_SERVER);
        History external = new History();
        external.setReference(History.REFERENCE_CLIENT);
        List<History> input = Arrays.asList(internal, external);
        List<History> out = History.getReferenceHistory(input, History.REFERENCE_CLIENT);
        Assert.assertNotNull(out);
        Assert.assertEquals(1, out.size());
        Assert.assertSame(external, out.get(0));
    }

    @Test
    public void getReferenceHistory_noMatch_returnsNull() {
        History h = new History();
        h.setReference(History.REFERENCE_SERVER);
        Assert.assertNull(History.getReferenceHistory(Collections.singletonList(h), History.REFERENCE_CLIENT));
    }

    @Test
    public void buildLastTimeline_nullHistories_returnsNull() {
        Assert.assertNull(History.buildLastTimeline(null));
    }

    @Test
    public void buildLastTimeline_emptyHistories_returnsNull() {
        Assert.assertNull(History.buildLastTimeline(Collections.emptyList()));
    }

    @Test
    public void buildLastTimeline_singleHistory_returnsItsCreated() {
        History h = new History();
        h.setCreated(1000L);
        Assert.assertEquals(Long.valueOf(1000L), History.buildLastTimeline(Collections.singletonList(h)));
    }

    @Test
    public void buildLastTimeline_multipleHistories_returnsMax() {
        History h1 = new History();
        h1.setCreated(100L);
        History h2 = new History();
        h2.setCreated(300L);
        History h3 = new History();
        h3.setCreated(200L);
        Assert.assertEquals(Long.valueOf(300L), History.buildLastTimeline(Arrays.asList(h1, h2, h3)));
    }

    @Test
    public void buildLastTimeline_allNullCreated_returnsNull() {
        History h1 = new History();
        h1.setCreated(null);
        History h2 = new History();
        h2.setCreated(null);
        Assert.assertNull(History.buildLastTimeline(Arrays.asList(h1, h2)));
    }

    @Test
    public void buildLastTimeline_someNullCreated_returnsMaxOfNonNull() {
        History h1 = new History();
        h1.setCreated(null);
        History h2 = new History();
        h2.setCreated(500L);
        History h3 = new History();
        h3.setCreated(200L);
        Assert.assertEquals(Long.valueOf(500L), History.buildLastTimeline(Arrays.asList(h1, h2, h3)));
    }

    @Test
    public void buildFirstTimeline_nullHistories_returnsNull() {
        Assert.assertNull(History.buildFirstTimeline(null));
    }

    @Test
    public void buildFirstTimeline_emptyHistories_returnsNull() {
        Assert.assertNull(History.buildFirstTimeline(Collections.emptyList()));
    }

    @Test
    public void buildFirstTimeline_singleHistory_returnsItsCreated() {
        History h = new History();
        h.setCreated(1000L);
        Assert.assertEquals(Long.valueOf(1000L), History.buildFirstTimeline(Collections.singletonList(h)));
    }

    @Test
    public void buildFirstTimeline_multipleHistories_returnsMin() {
        History h1 = new History();
        h1.setCreated(300L);
        History h2 = new History();
        h2.setCreated(100L);
        History h3 = new History();
        h3.setCreated(200L);
        Assert.assertEquals(Long.valueOf(100L), History.buildFirstTimeline(Arrays.asList(h1, h2, h3)));
    }

    @Test
    public void buildFirstTimeline_allNullCreated_returnsNull() {
        History h1 = new History();
        h1.setCreated(null);
        History h2 = new History();
        h2.setCreated(null);
        Assert.assertNull(History.buildFirstTimeline(Arrays.asList(h1, h2)));
    }

    @Test
    public void buildFirstTimeline_someNullCreated_returnsMinOfNonNull() {
        History h1 = new History();
        h1.setCreated(null);
        History h2 = new History();
        h2.setCreated(500L);
        History h3 = new History();
        h3.setCreated(200L);
        Assert.assertEquals(Long.valueOf(200L), History.buildFirstTimeline(Arrays.asList(h1, h2, h3)));
    }

    @Test
    public void getReferenceHistory_allMatch_returnsAll() {
        History a = new History();
        a.setReference(History.REFERENCE_CLIENT);
        History b = new History();
        b.setReference(History.REFERENCE_CLIENT);
        List<History> out = History.getReferenceHistory(Arrays.asList(a, b), History.REFERENCE_CLIENT);
        Assert.assertNotNull(out);
        Assert.assertEquals(2, out.size());
        Assert.assertTrue(out.contains(a));
        Assert.assertTrue(out.contains(b));
    }

    /** buildMarkdown：表头 + QUERY/ANSWER 行，content 与 created 落表 */
    @Test
    public void testBuildMarkdown_queryAndAnswerRows() throws Exception {
        History q = new History();
        q.setType(History.TYPE_QUERY);
        q.setContent("user asks");
        q.setCreated(10L);
        History a = new History();
        a.setType(History.TYPE_ANSWER);
        a.setContent("bot says");
        a.setCreated(20L);
        String md = History.buildMarkdown(Arrays.asList(q, a), null);
        Assert.assertTrue(md.contains("|The type field|The content of the current interaction|"));
        Assert.assertTrue(md.contains("|---|---|---|---|"));
        Assert.assertTrue(md.contains("|QUERY|user asks||10|"));
        Assert.assertTrue(md.contains("|ANSWER|bot says||20|"));
    }

    /** buildMarkdown：content 中的反斜杠会被去掉 */
    @Test
    public void testBuildMarkdown_stripsBackslashesInContent() throws Exception {
        History h = new History();
        h.setType(History.TYPE_QUERY);
        h.setContent("a\\b\\\\c");
        h.setCreated(1L);
        String md = History.buildMarkdown(Collections.singletonList(h), null);
        Assert.assertTrue(md.contains("|QUERY"));
    }

    /**
     * buildMarkdown：会先 {@link Collections#reverse} 入参列表，最新一条（原列表末尾）会排在表格最前。
     */
    @Test
    public void testBuildMarkdown_reversesList_orderNewestFirstInTable() throws Exception {
        History first = new History();
        first.setType(History.TYPE_QUERY);
        first.setContent("older");
        first.setCreated(100L);
        History second = new History();
        second.setType(History.TYPE_ANSWER);
        second.setContent("newer");
        second.setCreated(200L);
        ArrayList<History> list = new ArrayList<>(Arrays.asList(first, second));
        String md = History.buildMarkdown(list, null);
        int idxNewer = md.indexOf("|newer|");
        int idxOlder = md.indexOf("|older|");
        Assert.assertTrue(idxNewer > 0 && idxOlder > idxNewer);
    }

    /** buildMarkdown：会原地 reverse 传入的 List，调用方需注意副作用 */
    @Test
    public void testBuildMarkdown_reversesList_mutatesInputOrder() throws Exception {
        History a = new History();
        a.setType(History.TYPE_QUERY);
        a.setContent("a");
        a.setCreated(1L);
        History b = new History();
        b.setType(History.TYPE_ANSWER);
        b.setContent("b");
        b.setCreated(2L);
        ArrayList<History> list = new ArrayList<>(Arrays.asList(a, b));
        History.buildMarkdown(list, null);
        Assert.assertSame(b, list.get(1));
        Assert.assertSame(a, list.get(0));
    }

    /** truncate 为 null 时直接拼接表格体 */
    @Test
    public void testBuildMarkdown_truncateNull_noTransform() throws Exception {
        History h = new History();
        h.setType(History.TYPE_QUERY);
        h.setContent("x");
        h.setCreated(5L);
        String md = History.buildMarkdown(Collections.singletonList(h), null);
        Assert.assertTrue(md.endsWith(System.lineSeparator()));
        Assert.assertTrue(md.contains("|QUERY|x||5|"));
    }

    /** truncate 非空时对表格体字符串做后处理再写入 */
    @Test
    public void testBuildMarkdown_truncateNonNull_appliesTruncate() throws Exception {
        History h = new History();
        h.setType(History.TYPE_QUERY);
        h.setContent("body");
        h.setCreated(7L);
        HistoryTruncate truncate = body -> "[T]" + body;
        String md = History.buildMarkdown(Collections.singletonList(h), truncate);
        Assert.assertTrue(md.contains("[T]|QUERY|body||7|"));
    }

    @Test
    public void testIsApiIgnoreCase() {
        History h = new History();
        h.setApi(ProviderRequest.REQUEST_GOOGLE.toUpperCase());
        Assert.assertTrue(h.isApi(ProviderRequest.REQUEST_GOOGLE));
        Assert.assertTrue(h.isApi("Google"));
        Assert.assertFalse(h.isApi(ProviderRequest.REQUEST_OPENAI));
    }

    @Test
    public void testIsApiWithVarargsAnyMatchIgnoreCase() {
        History h = new History();
        h.setApi("AnThRoPiC");
        Assert.assertTrue(h.isApi(ProviderRequest.REQUEST_GOOGLE, ProviderRequest.REQUEST_ANTHROPIC));
        Assert.assertFalse(h.isApi(ProviderRequest.REQUEST_OPENAI, ProviderRequest.REQUEST_COZE));
    }

    @Test
    public void testIsEncryptForProtectedApisAndNormalApis() {
        History h = new History();
        h.setApi("google");
        Assert.assertTrue(h.isEncrypt());
        h.setApi("ANTHROPIC");
        Assert.assertTrue(h.isEncrypt());
        h.setApi(ProviderRequest.REQUEST_OPENAI);
        Assert.assertFalse(h.isEncrypt());
        h.setApi(null);
        Assert.assertFalse(h.isEncrypt());
    }

    @Test
    public void testBuildMarkdown_hidesReasoningForGoogleIgnoreCase() throws Exception {
        History h = new History();
        h.setType(History.TYPE_ANSWER);
        h.setApi("GOOGLE");
        h.setReason("hidden-reasoning");
        h.setContent("content");
        h.setCreated(1L);
        String md = History.buildMarkdown(Collections.singletonList(h), null);
        Assert.assertTrue(md.contains("|ANSWER|content||1|"));
        Assert.assertFalse(md.contains("hidden-reasoning"));
    }

    @Test
    public void testBuildMarkdown_showsReasoningForNonGoogle() throws Exception {
        History h = new History();
        h.setType(History.TYPE_ANSWER);
        h.setApi(ProviderRequest.REQUEST_OPENAI);
        h.setReason("show-reasoning");
        h.setContent("content");
        h.setCreated(2L);
        String md = History.buildMarkdown(Collections.singletonList(h), null);
        Assert.assertTrue(md.contains("|ANSWER|content|show-reasoning|2|"));
    }

    @Test
    public void testBuildMarkdown_hidesReasoningForAnthropicIgnoreCase() throws Exception {
        History h = new History();
        h.setType(History.TYPE_ANSWER);
        h.setApi("ANTHROPIC");
        h.setReason("hidden-anthropic-reasoning");
        h.setContent("content");
        h.setCreated(3L);
        String md = History.buildMarkdown(Collections.singletonList(h), null);
        Assert.assertTrue(md.contains("|ANSWER|content||3|"));
        Assert.assertFalse(md.contains("hidden-anthropic-reasoning"));
    }

    @Test
    public void testBuildMarkdown_showsReasoningWhenApiIsNull() throws Exception {
        History h = new History();
        h.setType(History.TYPE_ANSWER);
        h.setApi(null);
        h.setReason("show-when-api-null");
        h.setContent("content");
        h.setCreated(4L);
        String md = History.buildMarkdown(Collections.singletonList(h), null);
        Assert.assertTrue(md.contains("|ANSWER|content|show-when-api-null|4|"));
    }

    /** 空列表：仅有表头与空表格体区域 */
    @Test
    public void testBuildMarkdown_emptyHistories_headerOnly() throws Exception {
        String md = History.buildMarkdown(Collections.emptyList(), null);
        Assert.assertTrue(md.contains("|The type field|The content of the current interaction|"));
        Assert.assertTrue(md.contains("|---|---|---|"));
    }

    /**
     * History(WorkflowTask)：从任务拷贝 conversation、reference=SERVER、source=SplitUtils.join(task)、chat、created；
     * 其余字段保持类默认值（如 FUN_CHAT）。
     */
    @Test
    public void constructor_workTask_setsConversationReferenceSourceChatCreated() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nr = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nr.setConversation("conv-wt");
        nr.setChat("chat-wt");
        nr.setCreated(77_888L);
        nr.setWorkflow("workflowPlain");
        nr.setBiz("bizPlain");

        WorkflowTask task = nr;
        History h = new History(task);

        Assert.assertEquals("conv-wt", h.getConversation());
        Assert.assertEquals(History.REFERENCE_SERVER, h.getReference());
        Assert.assertEquals(SplitUtils.join(nr), h.getSource());
        Assert.assertEquals("chat-wt", h.getChat());
        Assert.assertEquals(Long.valueOf(77_888L), h.getCreated());
        Assert.assertEquals(History.FUN_CHAT, h.getFunction());
    }
}
