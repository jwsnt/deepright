package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.XmlUtils;
import ai.open.right.workflow.flow.command.QuickCommand;
import ai.open.right.workflow.flow.command.QuickCommandStore;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ExecutorService;

public class RagCmdTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = RagCmd.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagCmd.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void test() throws Exception {
        QuickCommand command = new QuickCommand();
        command.setPriority(100L);
        command.setContent("World");
        command.setCommand("UNKNOWN");
        List<QuickCommand> commands = new ArrayList<QuickCommand>();
        commands.add(command);
        QuickCommandStore qcs = EasyMock.createMock(QuickCommandStore.class);
        EasyMock.expect(qcs.restore("UNKNOWN", "UNKNOWN", "UNKNOWN")).andReturn(commands).anyTimes();
        EasyMock.replay(qcs);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setOverride(true);
        RagCmd cmd = new RagCmd();
        cmd.setQuickCommandStore(qcs);
        RagFuture future = cmd.rag(ragConfig, ragData);
        Assert.assertEquals(future.getClass(), RagAtOnce.class);
        Assert.assertEquals("World", ragData.getQuery().getQuery());
    }

    @Test
    public void testRag() throws Exception {
        QuickCommand command = new QuickCommand();
        command.setPriority(100L);
        command.setContent("Hello");
        command.setCommand("UNKNOWN");
        List<QuickCommand> commands = new ArrayList<QuickCommand>();
        commands.add(command);
        QuickCommandStore qcs = EasyMock.createMock(QuickCommandStore.class);
        EasyMock.expect(qcs.restore("UNKNOWN", "UNKNOWN", "UNKNOWN")).andReturn(commands).anyTimes();
        EasyMock.replay(qcs);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("WORLD")
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setOverride(true);
        RagCmd cmd = new RagCmd();
        cmd.setQuickCommandStore(qcs);
        cmd.rag(ragConfig, ragData);
        Assert.assertEquals("Hello", ragData.getQuery().getQuery());
    }

    @Test
    public void testWithConditionFailed() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagCmd cmd = new RagCmd();
        cmd.setNotifierService(notifierManager);
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        Assert.assertEquals(RagFuture.NOTHING, cmd.rag(ragConfig, ragData));
    }

    @Test
    public void testPriority() throws Exception {
        QuickCommand command1 = new QuickCommand();
        command1.setPriority(100L);
        command1.setContent("Hello");
        command1.setCommand("UNKNOWN");
        QuickCommand command2 = new QuickCommand();
        command2.setPriority(1001L);
        command2.setContent("World");
        command2.setCommand("UNKNOWN");
        QuickCommand command3 = new QuickCommand();
        command3.setPriority(1000L);
        command3.setContent("Today");
        command3.setCommand("UNKNOWN");
        List<QuickCommand> commands = new ArrayList<QuickCommand>();
        commands.add(command1);
        commands.add(command2);
        commands.add(command3);
        QuickCommandStore qcs = EasyMock.createMock(QuickCommandStore.class);
        EasyMock.expect(qcs.restore("UNKNOWN", "UNKNOWN", "UNKNOWN")).andReturn(commands).anyTimes();
        EasyMock.replay(qcs);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setOverride(true);
        RagCmd cmd = new RagCmd();
        cmd.setQuickCommandStore(qcs);
        RagFuture future = cmd.rag(ragConfig, ragData);
        Assert.assertEquals(future.getClass(), RagAtOnce.class);
        Assert.assertEquals("World", ragData.getQuery().getQuery());
    }

    @Test
    public void testXml() throws Exception {
        QuickCommand command1 = new QuickCommand();
        command1.setPriority(100L);
        command1.setContent("Hello");
        command1.setCommand("UNKNOWN1");
        QuickCommand command2 = new QuickCommand();
        command2.setPriority(1001L);
        command2.setContent("World");
        command2.setCommand("UNKNOWN1");
        QuickCommand command3 = new QuickCommand();
        command3.setPriority(1000L);
        command3.setContent("Today");
        command3.setCommand("UNKNOWN2");
        List<QuickCommand> commands = new ArrayList<QuickCommand>();
        commands.add(command1);
        commands.add(command2);
        commands.add(command3);
        RagCmd.LLMCommandsPrompts llmCommandsPrompts = new RagCmd.LLMCommandsPrompts(commands);
        Assert.assertEquals("<?xml version=\"1.0\" encoding=\"UTF-8\"?><Commands xmlns=\"\"><Command><UNKNOWN1>World</UNKNOWN1><UNKNOWN2>Today</UNKNOWN2></Command></Commands>", XmlUtils.write(llmCommandsPrompts));
    }

    @Test
    public void testReplace() throws Exception {
        QuickCommand command = new QuickCommand();
        command.setPriority(100L);
        command.setContent("Hello");
        command.setCommand("World");
        List<QuickCommand> commands = new ArrayList<QuickCommand>();
        commands.add(command);
        QuickCommandStore qcs = EasyMock.createMock(QuickCommandStore.class);
        EasyMock.expect(qcs.restore("UNKNOWN", "UNKNOWN", "UNKNOWN")).andReturn(commands).anyTimes();
        EasyMock.replay(qcs);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO #cmd")
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#cmd");
        ragConfig.setMode(RagConfig.MODE_XML);
        RagCmd cmd = new RagCmd();
        cmd.setQuickCommandStore(qcs);
        cmd.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", ragData.getQuery().getQuery());
        Assert.assertEquals("HELLO <?xml version=\"1.0\" encoding=\"UTF-8\"?><Commands xmlns=\"\"><Command><World>Hello</World></Command></Commands>", ragData.getPrompt());
    }

    @Test
    public void testReplaceWithJson() throws Exception {
        QuickCommand command = new QuickCommand();
        command.setPriority(100L);
        command.setContent("Hello");
        command.setCommand("World");
        List<QuickCommand> commands = new ArrayList<QuickCommand>();
        commands.add(command);
        QuickCommandStore qcs = EasyMock.createMock(QuickCommandStore.class);
        EasyMock.expect(qcs.restore("UNKNOWN", "UNKNOWN", "UNKNOWN")).andReturn(commands).anyTimes();
        EasyMock.replay(qcs);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO #cmd")
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#cmd");
        ragConfig.setMode(RagConfig.MODE_JSON);
        RagCmd cmd = new RagCmd();
        cmd.setQuickCommandStore(qcs);
        cmd.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", ragData.getQuery().getQuery());
        Assert.assertEquals("HELLO [{\"priority\":100,\"command\":\"World\",\"content\":\"Hello\"}]", ragData.getPrompt());
    }


    @Test
    public void testRagWithException() throws Exception {
        QuickCommandStore qcs = EasyMock.createMock(QuickCommandStore.class);
        EasyMock.expect(qcs.restore("UNKNOWN", "UNKNOWN", "UNKNOWN")).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(qcs);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("WORLD")
                .build();
        RagCmd cmd = new RagCmd();
        cmd.setQuickCommandStore(qcs);
        cmd.rag(new RagConfig(), ragData);
        Assert.assertEquals("UNKNOWN", ragData.getQuery().getQuery());
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        ExecutorService executorService = EasyMock.createMock(ExecutorService.class);
        QuickCommandStore quickCommand = EasyMock.createMock(QuickCommandStore.class);
        EasyMock.replay(executorService, quickCommand);
        RagCmd.InitConfig service = new RagCmd.InitConfig();
        service.setNotifierService(notifierManager);
        service.setQuickCommandStore(quickCommand);
        service.setTimeout4Condition(10086);
        RagCmd empty = service.ragCmd();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(quickCommand, empty.getQuickCommandStore());
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        EasyMock.verify(executorService, quickCommand);
    }
}
