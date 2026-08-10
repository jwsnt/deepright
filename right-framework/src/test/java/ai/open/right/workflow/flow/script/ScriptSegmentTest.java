package ai.open.right.workflow.flow.script;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.llm.LLMUsage;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentUsage;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.notify.Notifier;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;
import java.util.List;
import java.util.Map;

public class ScriptSegmentTest {

    @Test
    public void testGet() throws Exception {
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setWrap(ScriptConfig.WRAP_OBJECT);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        workflowTask.setNotifier(Notifier.SOURCE);
        ScriptSegment scriptSegment = new ScriptSegment(workflowTask, ScriptResponse.builder()
                .data("HELLO")
                .code(200)
                .build());
        Assert.assertEquals("UNKNOWN-UNKNOWN-UNKNOWN", scriptSegment.getDimension());
        Assert.assertEquals(workflowTask.getBiz(), scriptSegment.getBiz());
        Assert.assertTrue(scriptSegment.isFinished());
        Assert.assertEquals(workflowTask.getChat(), scriptSegment.getChat());
        Assert.assertEquals(Integer.valueOf(200), scriptSegment.getCode());
        Assert.assertEquals(workflowTask.getConversation(), scriptSegment.getConversation());
        Assert.assertEquals("HELLO", scriptSegment.getContent());
        Assert.assertEquals(Integer.valueOf(0), scriptSegment.getIndex());
        Assert.assertEquals("HELLO", scriptSegment.getData());
        Assert.assertEquals(workflowTask.getMetadata(), scriptSegment.getMetadata());
        Assert.assertEquals(workflowTask.getProtocol(), scriptSegment.getProtocol());
        Assert.assertEquals(Notifier.SOURCE, scriptSegment.getNotifier());
        Assert.assertEquals(workflowTask.getDeepness(), scriptSegment.getDeepness());
        Assert.assertEquals(Segment.ROLE_ANSWER, scriptSegment.getRole());
        Assert.assertEquals(false, scriptSegment.getSilent());
        Assert.assertEquals(false, scriptSegment.getStream());
        Assert.assertEquals(workflowTask.getCreated(), scriptSegment.getTimestamp());
        Assert.assertEquals("UNKNOWN@UNKNOWN", scriptSegment.getUpstream());
        Assert.assertEquals(workflowTask.getPrevious(), scriptSegment.getPrevious());
        Assert.assertEquals(workflowTask.getTrace(), scriptSegment.getTrace());
        Assert.assertEquals(workflowTask.getWorkflow(), scriptSegment.getWorkflow());
        Assert.assertEquals(workflowTask.getOriginal(), scriptSegment.getOriginal());
        Assert.assertEquals(workflowTask.getUserContext(), scriptSegment.getUserContext());
        Assert.assertEquals("INITIAL", scriptSegment.getInitial());
        Assert.assertEquals(workflowTask.isEntry(), scriptSegment.isEntry());
        UserContext userContext = UserContext.builder().build();
        scriptSegment.setUserContext(userContext);
        Assert.assertEquals(userContext, scriptSegment.getUserContext());
        LLMUsage nettyUsage = scriptSegment.getUsage();
        scriptSegment.setUsage(new SegmentUsage(TokenData.builder().cache(1).total(1).build()));
        Assert.assertEquals(nettyUsage, scriptSegment.getUsage());

    }

    @Test
    public void testSet() throws Exception {
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setWrap(ScriptConfig.WRAP_OBJECT);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        ScriptSegment scriptSegment = new ScriptSegment(workflowTask, ScriptResponse.builder()
                .data("HELLO")
                .code(200)
                .build());
        scriptSegment.setMetadata("HELLO", "WORLD");
        scriptSegment.setContent("HELLO1 WORLD2");
        scriptSegment.setNotifier("LOCALHOST");
        scriptSegment.setProtocol("SCRIPT");
        scriptSegment.setUpstream("UPSTREAM2");
        scriptSegment.setSilent(true);
        scriptSegment.setBiz("BIZ");
        scriptSegment.setWorkflow("WORKFLOW2");
        Assert.assertEquals("BIZ", scriptSegment.getBiz());
        Assert.assertEquals("WORLD", scriptSegment.getMetadata().get("HELLO"));
        Assert.assertEquals("HELLO1 WORLD2", scriptSegment.getContent());
        Assert.assertEquals("LOCALHOST", scriptSegment.getNotifier());
        Assert.assertEquals("SCRIPT", scriptSegment.getProtocol());
        Assert.assertEquals("UPSTREAM2", scriptSegment.getUpstream());
        Assert.assertEquals("WORKFLOW2", scriptSegment.getWorkflow());
        Assert.assertTrue(scriptSegment.getSilent());
    }

    @Test
    public void testSetDeepnessDelegatesToSegment() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        ScriptSegment scriptSegment = new ScriptSegment(workflowTask, ScriptResponse.builder().data("HELLO").code(200).build());
        scriptSegment.setDeepness(2);
        Assert.assertEquals("setDeepness should delegate to segment", Integer.valueOf(2), workflowTask.getDeepness());
        Assert.assertEquals(Integer.valueOf(2), scriptSegment.getDeepness());
    }

    @Test
    public void testWithDelegate() throws Exception {
        Segment segment = EasyMock.createMock(Segment.class);
        segment.init();
        EasyMock.expectLastCall().times(1);
        segment.mark();
        EasyMock.expectLastCall().times(1);
        segment.reset(false, 10);
        EasyMock.expectLastCall().times(1);
        segment.setMetadata(EasyMock.anyObject(Map.class));
        EasyMock.expectLastCall().times(1);
        segment.delMetadata();
        EasyMock.expectLastCall().times(1);
        EasyMock.replay(segment);
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setWrap(ScriptConfig.WRAP_OBJECT);
        ScriptSegment scriptSegment = new ScriptSegment(segment, "HELLO");
        scriptSegment.init();
        scriptSegment.mark();
        scriptSegment.reset(false, 10);
        scriptSegment.setMetadata(Collections.singletonMap("HELLO", "WORLD"));
        scriptSegment.delMetadata();
        EasyMock.verify(segment);
    }


    @Test
    public void testData() throws Exception {
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setWrap(ScriptConfig.WRAP_OBJECT);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        ScriptSegment scriptSegment = new ScriptSegment(workflowTask, ScriptResponse.builder()
                .data("你的输入大于50")
                .code(200)
                .build());
        Assert.assertEquals("你的输入大于50", scriptSegment.getData());
    }

    @Test
    public void testClone1() throws Exception {
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setWrap(ScriptConfig.WRAP_OBJECT);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        workflowTask.setNotifier(null);
        ScriptSegment segment = new ScriptSegment(workflowTask, ScriptResponse.builder()
                .data("HELLO")
                .code(200)
                .build());
        Segment scriptSegment = segment.copyWithWorkflow("WORKFLOW2");
        Assert.assertEquals(workflowTask.getBiz(), scriptSegment.getBiz());
        Assert.assertTrue(scriptSegment.isFinished());
        Assert.assertEquals(workflowTask.getChat(), scriptSegment.getChat());
        Assert.assertEquals(Integer.valueOf(200), scriptSegment.getCode());
        Assert.assertEquals(workflowTask.getConversation(), scriptSegment.getConversation());
        Assert.assertEquals("HELLO", scriptSegment.getContent());
        Assert.assertEquals(Integer.valueOf(0), scriptSegment.getIndex());
        Assert.assertEquals(workflowTask.getMetadata(), scriptSegment.getMetadata());
        Assert.assertEquals(workflowTask.getProtocol(), scriptSegment.getProtocol());
        Assert.assertEquals("endpoint", scriptSegment.getNotifier());
        Assert.assertEquals(workflowTask.getDeepness(), scriptSegment.getDeepness());
        Assert.assertEquals(Segment.ROLE_ANSWER, scriptSegment.getRole());
        Assert.assertEquals(false, scriptSegment.getSilent());
        Assert.assertEquals(false, scriptSegment.getStream());
        Assert.assertEquals(workflowTask.getCreated(), scriptSegment.getTimestamp());
        Assert.assertEquals("UNKNOWN@UNKNOWN", scriptSegment.getUpstream());
        Assert.assertEquals(workflowTask.getPrevious(), scriptSegment.getPrevious());
        Assert.assertEquals(workflowTask.getTrace(), scriptSegment.getTrace());
        Assert.assertEquals("WORKFLOW2", scriptSegment.getWorkflow());
        Assert.assertEquals(workflowTask.getUserContext().getDevice(), scriptSegment.getDevice());
        Assert.assertEquals(workflowTask.getOriginal(), scriptSegment.getOriginal());
        Assert.assertEquals(workflowTask.getUserContext(), scriptSegment.getUserContext());
        Assert.assertEquals("INITIAL", scriptSegment.getInitial());
    }

    @Test
    public void testClone2() throws Exception {
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setWrap(ScriptConfig.WRAP_OBJECT);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        ScriptSegment segment = new ScriptSegment(workflowTask, ScriptResponse.builder()
                .data("HELLO")
                .code(200)
                .build());
        Segment scriptSegment = segment.copy();
        Assert.assertEquals(workflowTask.getBiz(), scriptSegment.getBiz());
        Assert.assertTrue(scriptSegment.isFinished());
        Assert.assertEquals(workflowTask.getChat(), scriptSegment.getChat());
        Assert.assertEquals(Integer.valueOf(200), scriptSegment.getCode());
        Assert.assertEquals(workflowTask.getConversation(), scriptSegment.getConversation());
        Assert.assertEquals("HELLO", scriptSegment.getContent());
        Assert.assertEquals(Integer.valueOf(0), scriptSegment.getIndex());
        Assert.assertEquals(workflowTask.getMetadata(), scriptSegment.getMetadata());
        Assert.assertEquals(workflowTask.getProtocol(), scriptSegment.getProtocol());
        Assert.assertEquals("localhost", scriptSegment.getNotifier());
        Assert.assertEquals(workflowTask.getDeepness(), scriptSegment.getDeepness());
        Assert.assertEquals(Segment.ROLE_ANSWER, scriptSegment.getRole());
        Assert.assertEquals(false, scriptSegment.getSilent());
        Assert.assertEquals(false, scriptSegment.getStream());
        Assert.assertEquals(workflowTask.getCreated(), scriptSegment.getTimestamp());
        Assert.assertEquals("UNKNOWN@UNKNOWN", scriptSegment.getUpstream());
        Assert.assertEquals(workflowTask.getPrevious(), scriptSegment.getPrevious());
        Assert.assertEquals(workflowTask.getTrace(), scriptSegment.getTrace());
        Assert.assertEquals("UNKNOWN", scriptSegment.getWorkflow());
        Assert.assertEquals(workflowTask.getUserContext().getDevice(), scriptSegment.getDevice());
        Assert.assertEquals(workflowTask.getOriginal(), scriptSegment.getOriginal());
        Assert.assertEquals(workflowTask.getUserContext(), scriptSegment.getUserContext());
        Assert.assertEquals("INITIAL", scriptSegment.getInitial());
    }

    @Test
    public void testClone3() throws Exception {
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setWrap(ScriptConfig.WRAP_OBJECT);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        ScriptSegment segment = new ScriptSegment(workflowTask, ScriptResponse.builder()
                .data("HELLO")
                .code(200)
                .build());
        Segment scriptSegment = segment.copyWithNotifier("NOTIFIER");
        Assert.assertEquals("NOTIFIER", scriptSegment.getNotifier());
    }

    @Test
    public void testStart() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        ScriptSegment segment = new ScriptSegment(workflowTask, ScriptResponse.builder()
                .data("HELLO")
                .code(200)
                .build());
        Assert.assertEquals(Integer.valueOf(0), segment.getStart());
        segment.setStart(10);
        Assert.assertEquals(Integer.valueOf(10), segment.getStart());
        Segment segment2 = segment.copyWithStart(2);
        Assert.assertEquals(Integer.valueOf(2), segment2.getStart());
    }

    @Test
    public void testConstructorNoData() throws Exception {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ScriptResponse response = ScriptResponse.builder().code(200).build();
        ScriptSegment segment = new ScriptSegment(task, response);
        Assert.assertEquals("", segment.getContent());
    }

    @Test
    public void testCopyMetadata() throws Exception {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ScriptSegment segment = new ScriptSegment(task, ScriptResponse.builder().code(200).build());
        segment.setMetadata("K", "V");
        Segment copy = segment.copy();
        Assert.assertEquals("V", copy.getMetadata().get("K"));
    }

    @Test
    public void testPutMetadataDelegatesReference() throws Exception {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ScriptSegment scriptSegment = new ScriptSegment(task, ScriptResponse.builder().code(200).build());
        Map<String, Object> metadata = Collections.singletonMap("K", "V");

        scriptSegment.putMetadata(metadata);

        Assert.assertSame(metadata, scriptSegment.getMetadata());
    }

    @Test
    public void testCopyWithIdKeepsSameId() throws Exception {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ScriptSegment segment = new ScriptSegment(task, ScriptResponse.builder().data("HELLO").code(200).build());

        Segment copyWithId = segment.copyWithId();
        Segment copy = segment.copy();

        Assert.assertEquals(segment.getId(), copyWithId.getId());
        Assert.assertNotSame(segment, copyWithId);
        Assert.assertNotEquals(segment.getId(), copy.getId());
    }

    @Test
    public void testIncrDeepness_delegatesToSegment() {
        Segment segment = EasyMock.createMock(Segment.class);
        EasyMock.expect(segment.incrDeepness()).andReturn(segment).anyTimes();
        EasyMock.replay(segment);
        ScriptSegment scriptSegment = new ScriptSegment(segment, "HELLO");
        ScriptSegment result = scriptSegment.incrDeepness();
        Assert.assertSame("incrDeepness() 应返回 this 以支持链式调用", scriptSegment, result);
        EasyMock.verify(segment);
    }

    @Test
    public void isFromFunMerge_delegatesToWrappedSegment() {
        Segment segment = EasyMock.createMock(Segment.class);
        EasyMock.expect(segment.isFromFunMerge()).andReturn(true).once();
        EasyMock.replay(segment);
        Assert.assertTrue(new ScriptSegment(segment, "x").isFromFunMerge());
        EasyMock.verify(segment);
    }

    @Test
    public void isFromFunCall_delegatesToWrappedSegment() {
        Segment segment = EasyMock.createMock(Segment.class);
        EasyMock.expect(segment.isFromFunCall()).andReturn(true).once();
        EasyMock.replay(segment);
        Assert.assertTrue(new ScriptSegment(segment, "x").isFromFunCall());
        EasyMock.verify(segment);
    }

    @Test
    public void getHistories_delegatesToWrappedSegment() {
        Segment segment = EasyMock.createMock(Segment.class);
        List<History> list = Collections.singletonList(new History());
        EasyMock.expect(segment.getHistories()).andReturn(list).once();
        EasyMock.replay(segment);
        Assert.assertSame(list, new ScriptSegment(segment, "x").getHistories());
        EasyMock.verify(segment);
    }
}
