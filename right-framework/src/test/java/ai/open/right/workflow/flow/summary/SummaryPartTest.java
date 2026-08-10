package ai.open.right.workflow.flow.summary;

import ai.open.right.ObjectBuilder;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.List;

public class SummaryPartTest {

    private static final Long LAST_TIMELINE = 999L;

    private WorkflowTask buildWorkTask() {
        NettyRequest workTask = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        workTask.setConversation("conversation-x");
        workTask.setWorkflow("workflow-x");
        workTask.setBiz("biz-x");
        workTask.setChat("chat-x");
        return workTask;
    }

    @Test
    public void test() {
        SummaryPart summaryPart = SummaryPart.builder().build();
        Assert.assertFalse(summaryPart.hasPairs());
        summaryPart.setPairs(List.of(new HistoryPair()));
        Assert.assertTrue(summaryPart.hasPairs());
    }

    @Test
    public void testHasContentWhenContentNull() {
        SummaryPart part = SummaryPart.builder().content(null).build();
        Assert.assertFalse(part.hasContent());
    }

    @Test
    public void testHasContentWhenContentEmpty() {
        SummaryPart part = SummaryPart.builder().content("").build();
        Assert.assertFalse(part.hasContent());
    }

    @Test
    public void testHasContentWhenContentNonEmpty() {
        SummaryPart part = SummaryPart.builder().content("some content").build();
        Assert.assertTrue(part.hasContent());
    }

    @Test
    public void testHasContentWhenContentSetAfterBuild() {
        SummaryPart part = SummaryPart.builder().build();
        Assert.assertFalse(part.hasContent());
        part.setContent("added");
        Assert.assertTrue(part.hasContent());
        part.setContent("");
        Assert.assertFalse(part.hasContent());
    }

    @Test
    public void testInitShouldFilterAssistantPairWhenAnswerEmpty() {
        HistoryPair assistantEmptyAnswer = new HistoryPair();
        assistantEmptyAnswer.setRole(History.ROLE_ASSISTANT);
        assistantEmptyAnswer.setAnswer("");
        assistantEmptyAnswer.setQuery("ignored");

        HistoryPair userValid = new HistoryPair();
        userValid.setRole(History.ROLE_USER);
        userValid.setQuery("user query");

        SummaryPart part = SummaryPart.builder()
                .pairs(new ArrayList<>(List.of(assistantEmptyAnswer, userValid)))
                .build();

        SummaryPart returned = part.init(buildWorkTask(), LAST_TIMELINE);

        Assert.assertSame(part, returned);
        Assert.assertEquals(1, part.getPairs().size());
        Assert.assertSame(userValid, part.getPairs().get(0));
    }

    @Test
    public void testInitShouldFilterUserPairWhenQueryEmpty() {
        HistoryPair userEmptyQuery = new HistoryPair();
        userEmptyQuery.setRole(History.ROLE_USER);
        userEmptyQuery.setQuery("");
        userEmptyQuery.setAnswer("ignored");

        HistoryPair assistantValid = new HistoryPair();
        assistantValid.setRole(History.ROLE_ASSISTANT);
        assistantValid.setAnswer("assistant answer");

        SummaryPart part = SummaryPart.builder()
                .pairs(new ArrayList<>(List.of(userEmptyQuery, assistantValid)))
                .build();

        part.init(buildWorkTask(), LAST_TIMELINE);

        Assert.assertEquals(1, part.getPairs().size());
        Assert.assertSame(assistantValid, part.getPairs().get(0));
    }

    @Test
    public void testInitShouldKeepPairsWhenRequiredFieldsPresent() {
        HistoryPair assistantValid = new HistoryPair();
        assistantValid.setRole(History.ROLE_ASSISTANT);
        assistantValid.setAnswer("assistant answer");

        HistoryPair userValid = new HistoryPair();
        userValid.setRole(History.ROLE_USER);
        userValid.setQuery("user query");

        SummaryPart part = SummaryPart.builder()
                .pairs(new ArrayList<>(List.of(assistantValid, userValid)))
                .build();

        part.init(buildWorkTask(), LAST_TIMELINE);

        Assert.assertEquals(2, part.getPairs().size());
        Assert.assertSame(assistantValid, part.getPairs().get(0));
        Assert.assertSame(userValid, part.getPairs().get(1));
    }

    @Test
    public void testInitShouldReturnSelfWhenPairsNullOrEmpty() {
        SummaryPart partWithNullPairs = SummaryPart.builder().build();
        Assert.assertSame(partWithNullPairs, partWithNullPairs.init(buildWorkTask(), LAST_TIMELINE));
        Assert.assertNull(partWithNullPairs.getPairs());

        SummaryPart partWithEmptyPairs = SummaryPart.builder()
                .pairs(new ArrayList<>())
                .build();
        Assert.assertSame(partWithEmptyPairs, partWithEmptyPairs.init(buildWorkTask(), LAST_TIMELINE));
        Assert.assertTrue(partWithEmptyPairs.getPairs().isEmpty());
    }

    @Test
    public void testInitShouldFilterPairsWhenRequiredFieldsAreNull() {
        HistoryPair assistantNullAnswer = new HistoryPair();
        assistantNullAnswer.setRole(History.ROLE_ASSISTANT);
        assistantNullAnswer.setAnswer(null);

        HistoryPair userNullQuery = new HistoryPair();
        userNullQuery.setRole(History.ROLE_USER);
        userNullQuery.setQuery(null);

        HistoryPair validPair = new HistoryPair();
        validPair.setRole(History.ROLE_USER);
        validPair.setQuery("valid query");

        SummaryPart part = SummaryPart.builder()
                .pairs(new ArrayList<>(List.of(assistantNullAnswer, userNullQuery, validPair)))
                .build();

        part.init(buildWorkTask(), LAST_TIMELINE);

        Assert.assertEquals(1, part.getPairs().size());
        Assert.assertSame(validPair, part.getPairs().get(0));
    }

    @Test
    public void testInitShouldFillMissingFieldsFromWorkflowTask() {
        WorkflowTask workTask = buildWorkTask();
        HistoryPair pair = new HistoryPair();
        pair.setRole(History.ROLE_USER);
        pair.setQuery("user query");
        pair.setConversation("");
        pair.setSource("");
        pair.setChat("");
        pair.setModel("");
        pair.setApi("");
        pair.setCreated(null);

        SummaryPart part = SummaryPart.builder()
                .pairs(new ArrayList<>(List.of(pair)))
                .build();

        SummaryPart returned = part.init(workTask, LAST_TIMELINE);

        Assert.assertSame(part, returned);
        Assert.assertEquals(workTask.getConversation(), pair.getConversation());
        Assert.assertEquals(SplitUtils.join(workTask), pair.getSource());
        Assert.assertEquals(workTask.getChat(), pair.getChat());
        Assert.assertEquals(LLMQueryService.LLM_UNKNOW, pair.getModel());
        Assert.assertEquals(LLMQueryService.LLM_UNKNOW, pair.getApi());
        Assert.assertEquals(LAST_TIMELINE, pair.getCreated());
    }

    @Test
    public void testInitShouldKeepExistingFieldsWhenPresent() {
        WorkflowTask workTask = buildWorkTask();
        HistoryPair pair = new HistoryPair();
        pair.setRole(History.ROLE_ASSISTANT);
        pair.setAnswer("assistant answer");
        pair.setConversation("conversation-y");
        pair.setSource("biz-y@workflow-y");
        pair.setChat("chat-y");
        pair.setModel("model-y");
        pair.setApi("api-y");
        pair.setCreated(123L);

        SummaryPart part = SummaryPart.builder()
                .pairs(new ArrayList<>(List.of(pair)))
                .build();

        part.init(workTask, LAST_TIMELINE);

        Assert.assertEquals("conversation-y", pair.getConversation());
        Assert.assertEquals("biz-y@workflow-y", pair.getSource());
        Assert.assertEquals("chat-y", pair.getChat());
        Assert.assertEquals("model-y", pair.getModel());
        Assert.assertEquals("api-y", pair.getApi());
        Assert.assertEquals(Long.valueOf(123L), pair.getCreated());
    }
}
