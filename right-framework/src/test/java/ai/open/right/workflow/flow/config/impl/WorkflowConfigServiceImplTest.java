package ai.open.right.workflow.flow.config.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.Config;
import ai.open.right.workflow.config.ConfigService;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.mcp.McpToolsCallAssistant;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class WorkflowConfigServiceImplTest {

    @Test
    public void testConfig() throws Exception {
        ConfigService config = EasyMock.createMock(ConfigService.class);
        Config cf = new Config("BIZ", new HashMap<>());
        Map<String, Object> mp = new HashMap<String, Object>();
        Map<String, Object> io = new HashMap<String, Object>();
        io.put("assistant", "X");
        mp.put("HELLO", io);
        cf.setConfigs(mp);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setWorkflow("HELLO");
        workflowTask.setBiz("BIZ");
        EasyMock.expect(config.get(EasyMock.anyObject())).andReturn(cf).anyTimes();
        EasyMock.replay(config);
        WorkflowConfigServiceImpl service = new WorkflowConfigServiceImpl();
        service.setNamesService(ObjectBuilder.buildNamesService());
        service.setConfigService(config);
        WorkflowConfig wc = service.config(workflowTask);
        Assert.assertEquals("X", wc.getAssistant());
        Assert.assertNotNull(wc.getLlmConfig());
        Assert.assertEquals("HELLO", wc.getLlmConfig().getPrompt());
        EasyMock.verify(config);
    }

    @Test
    public void testConfigWithProvider1() throws Exception {
        ConfigService config = EasyMock.createMock(ConfigService.class);
        Config cf = new Config("BIZ", new HashMap<>());
        Map<String, Object> mp = new HashMap<String, Object>();
        Map<String, Object> io = new HashMap<String, Object>();
        io.put("assistant", "X");
        mp.put("C", io);
        cf.setConfigs(mp);
        EasyMock.expect(config.get(EasyMock.anyObject())).andReturn(cf).anyTimes();
        EasyMock.replay(config);
        WorkflowConfigServiceImpl service = new WorkflowConfigServiceImpl();
        service.setNamesService(ObjectBuilder.buildNamesService());
        service.setConfigService(config);
        service.setProvider("HELLO");
        WorkflowConfig wc = service.config(ObjectBuilder.buildWorkflowTask());
        Assert.assertEquals("HELLO", wc.getLlmConfig().getProvider());
    }

    @Test
    public void testConfigWithProvider2() throws Exception {
        ConfigService config = EasyMock.createMock(ConfigService.class);
        Config cf = new Config("BIZ", new HashMap<>());
        Map<String, Object> mp = new HashMap<String, Object>();
        Map<String, Object> io = new HashMap<String, Object>();
        io.put("assistant", "X");
        io.put("llm", new HashMap<>());
        mp.put("C", io);
        cf.setConfigs(mp);
        EasyMock.expect(config.get(EasyMock.anyObject())).andReturn(cf).anyTimes();
        EasyMock.replay(config);
        WorkflowConfigServiceImpl service = new WorkflowConfigServiceImpl();
        service.setNamesService(ObjectBuilder.buildNamesService());
        service.setConfigService(config);
        service.setProvider("HELLO");
        WorkflowConfig wc = service.config(ObjectBuilder.buildWorkflowTask());
        Assert.assertEquals("HELLO", wc.getLlmConfig().getProvider());
    }

    @Test
    public void testMcp() throws Exception {
        WorkflowConfigServiceImpl service = new WorkflowConfigServiceImpl();
        service.setNamesService(ObjectBuilder.buildNamesService());
        WorkflowTask workflow = ObjectBuilder.buildWorkflowTask();
        workflow.setWorkflow(NamesService.PREFIX_TOOLS + "C");
        WorkflowConfig wc = service.config(workflow);
        Assert.assertEquals(McpToolsCallAssistant.WORKFLOW_NAME, wc.getAssistant());
    }

    @Test
    public void testDefConfigWithOutProvider() throws Exception {
        ConfigService config = EasyMock.createMock(ConfigService.class);
        Config cf = new Config("BIZ", new HashMap<>());
        Map<String, Object> mp = new HashMap<String, Object>();
        Map<String, Object> io = new HashMap<String, Object>();
        io.put("llm", new HashMap<>());
        io.put("assistant", "X");
        mp.put("HELLO", io);
        cf.setConfigs(mp);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setWorkflow("HELLO");
        workflowTask.setBiz("BIZ");
        EasyMock.expect(config.get(EasyMock.anyObject())).andReturn(cf).anyTimes();
        EasyMock.replay(config);
        WorkflowConfigServiceImpl service = new WorkflowConfigServiceImpl();
        service.setNamesService(ObjectBuilder.buildNamesService());
        service.setProvider("PROVIDER1");
        service.setConfigService(config);
        WorkflowConfig wc = service.config(workflowTask);
        Assert.assertNotNull(wc.getLlmConfig());
        Assert.assertEquals("PROVIDER1", wc.getLlmConfig().getProvider());
        EasyMock.verify(config);
    }

    @Test
    public void testConfigWithPrefix() throws Exception {
        ConfigService config = EasyMock.createMock(ConfigService.class);
        Config cf = new Config("BIZ", new HashMap<>());
        Map<String, Object> mp = new HashMap<String, Object>();
        Map<String, Object> io = new HashMap<String, Object>();
        io.put("assistant", "X");
        mp.put("HELLO", io);
        cf.setConfigs(mp);
        NamesService namesService = ObjectBuilder.buildNamesService();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setWorkflow(namesService.encode(NamesService.PREFIX_WORKFLOW, "BIZ", "HELLO"));
        workflowTask.setBiz("BIZ");
        EasyMock.expect(config.get(EasyMock.anyObject())).andReturn(cf).anyTimes();
        EasyMock.replay(config);
        WorkflowConfigServiceImpl service = new WorkflowConfigServiceImpl();
        service.setNamesService(namesService);
        service.setConfigService(config);
        WorkflowConfig wc = service.config(workflowTask);
        Assert.assertEquals("X", wc.getAssistant());
        Assert.assertNotNull(wc.getLlmConfig());
        Assert.assertEquals("HELLO", wc.getLlmConfig().getPrompt());
        EasyMock.verify(config);
    }

    @Test
    public void testInit() throws Exception {
        ConfigService configService = EasyMock.createMock(ConfigService.class);
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.replay(configService, namesService);
        WorkflowConfigServiceImpl.InitConfig service = new WorkflowConfigServiceImpl.InitConfig();
        service.setConfigService(configService);
        service.setNamesService(namesService);
        service.setProvider("PROVIDER");
        WorkflowConfigServiceImpl empty = (WorkflowConfigServiceImpl) service.workflowConfigService();
        Assert.assertEquals(configService, empty.getConfigService());
        Assert.assertEquals(namesService, empty.getNamesService());
        Assert.assertEquals("PROVIDER", empty.getProvider());
    }

    @Test
    public void testConfigWithDefStream() throws Exception {
        ConfigService config = EasyMock.createMock(ConfigService.class);
        Config cf = new Config("BIZ", new HashMap<>());
        Map<String, Object> mp = new HashMap<String, Object>();
        Map<String, Object> io = new HashMap<String, Object>();
        io.put("assistant", "X");
        mp.put("C", io);
        cf.setConfigs(mp);
        EasyMock.expect(config.get(EasyMock.anyObject())).andReturn(cf).anyTimes();
        EasyMock.replay(config);
        WorkflowConfigServiceImpl service = new WorkflowConfigServiceImpl();
        service.setNamesService(ObjectBuilder.buildNamesService());
        service.setConfigService(config);
        service.setProvider("HELLO");
        WorkflowConfig wc = service.config(ObjectBuilder.buildWorkflowTask());
        Assert.assertTrue(wc.getLlmConfig().getStream());
    }

    // ---------- configExtended 单测 ----------

    /** 无 extended 时不做任何合并，config 不被调用 */
    @Test
    public void testConfigExtended_noExtended() throws Exception {
        WorkflowConfig target = new WorkflowConfig();
        // 不设置 extended，hasExtended() 为 false
        final List<String[]> configInvocations = new ArrayList<>();
        WorkflowConfigServiceImpl service = new WorkflowConfigServiceImpl() {
            @Override
            public WorkflowConfig config(String biz, String workflow) throws Exception {
                configInvocations.add(new String[]{biz, workflow});
                WorkflowConfig other = new WorkflowConfig();
                other.setLlmConfig(new LLMConfig());
                return other;
            }
        };
        service.configExtended(target, "BIZ");
        Assert.assertTrue("无 extended 时不应调用 config", configInvocations.isEmpty());
        Assert.assertNull(target.getLlmConfig());
    }

    /** extended 为单段且无 @ 时，SplitUtils.split(scene, biz) 得到 [biz, scene]，调用一次 config 并 merge */
    @Test
    public void testConfigExtended_singleSegmentNoAt() throws Exception {
        WorkflowConfig target = new WorkflowConfig();
        target.setExtended("BIZ");
        WorkflowConfig merged = new WorkflowConfig();
        merged.setLlmConfig(new LLMConfig());
        final List<String[]> invocations = new ArrayList<>();
        WorkflowConfigServiceImpl service = new WorkflowConfigServiceImpl() {
            @Override
            public WorkflowConfig config(String biz, String workflow) throws Exception {
                invocations.add(new String[]{biz, workflow});
                return merged;
            }
        };
        service.configExtended(target, "BIZ");
        Assert.assertEquals(1, invocations.size());
        Assert.assertEquals("BIZ", invocations.get(0)[0]);
        Assert.assertEquals("BIZ", invocations.get(0)[1]);
        Assert.assertEquals(merged.getLlmConfig(), target.getLlmConfig());
    }

    /** extended 为 biz@workflow 格式时，按 part[0]=biz、part[1]=workflow 调用 config 并 merge */
    @Test
    public void testConfigExtended_bizAtWorkflowFormat() throws Exception {
        WorkflowConfig target = new WorkflowConfig();
        target.setExtended("baseBiz@baseWorkflow");
        WorkflowConfig fromExtended = new WorkflowConfig();
        fromExtended.setAssistant("fromExtended");
        final List<String[]> invocations = new ArrayList<>();
        WorkflowConfigServiceImpl service = new WorkflowConfigServiceImpl() {
            @Override
            public WorkflowConfig config(String biz, String workflow) throws Exception {
                invocations.add(new String[]{biz, workflow});
                return fromExtended;
            }
        };
        service.configExtended(target, "currentBiz");
        Assert.assertEquals(1, invocations.size());
        Assert.assertEquals("baseBiz", invocations.get(0)[0]);
        Assert.assertEquals("baseWorkflow", invocations.get(0)[1]);
        Assert.assertEquals("fromExtended", target.getAssistant());
    }

    /** extended 为多段（逗号分隔）时，每段解析并依次 config + merge */
    @Test
    public void testConfigExtended_multipleSegments() throws Exception {
        WorkflowConfig target = new WorkflowConfig();
        target.setExtended("biz1@wf1,biz2@wf2");
        final List<String[]> invocations = new ArrayList<>();
        WorkflowConfig first = new WorkflowConfig();
        first.setAssistant("first");
        WorkflowConfig second = new WorkflowConfig();
        second.setLlmConfig(new LLMConfig());
        WorkflowConfigServiceImpl service = new WorkflowConfigServiceImpl() {
            @Override
            public WorkflowConfig config(String biz, String workflow) throws Exception {
                invocations.add(new String[]{biz, workflow});
                if ("biz1".equals(biz) && "wf1".equals(workflow)) {
                    return first;
                }
                if ("biz2".equals(biz) && "wf2".equals(workflow)) {
                    return second;
                }
                return new WorkflowConfig();
            }
        };
        service.configExtended(target, "anyBiz");
        Assert.assertEquals(2, invocations.size());
        Assert.assertEquals("biz1", invocations.get(0)[0]);
        Assert.assertEquals("wf1", invocations.get(0)[1]);
        Assert.assertEquals("biz2", invocations.get(1)[0]);
        Assert.assertEquals("wf2", invocations.get(1)[1]);
        Assert.assertEquals("first", target.getAssistant());
        Assert.assertEquals(second.getLlmConfig(), target.getLlmConfig());
    }
    @Test
    public void testBuildMcpConfigUnknown() throws Exception {
        WorkflowConfigServiceImpl service = new WorkflowConfigServiceImpl();
        service.setNamesService(ObjectBuilder.buildNamesService());
        Assert.assertNull(service.buildMcpConfig("UNKNOWN"));
    }
}
