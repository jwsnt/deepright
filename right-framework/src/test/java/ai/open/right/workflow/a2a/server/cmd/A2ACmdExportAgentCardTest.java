package ai.open.right.workflow.a2a.server.cmd;

import ai.open.right.ObjectBuilder;
import ai.open.right.netty.NettyAlarm;
import ai.open.right.netty.a2a.server.NettyA2ARequest;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.a2a.A2ARequest;
import ai.open.right.workflow.a2a.protocol.AgentCard;
import ai.open.right.workflow.config.ConfigSearch;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import com.google.common.collect.ImmutableMap;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.ByteBufAllocator;
import io.netty.channel.Channel;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.util.Attribute;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.net.SocketAddress;
import java.util.Map;

public class A2ACmdExportAgentCardTest {

    @Test
    public void testSize() throws Exception {
        A2ACmdExportAgentCard agentCard = new A2ACmdExportAgentCard() {
            @Override
            protected WorkflowConfig findWorkflowConfig(String[] pair) throws Exception {
                WorkflowConfig workflowConfig = new WorkflowConfig();
                LLMConfig llmConfig = new LLMConfig();
                llmConfig.setStream(true);
                workflowConfig.setLlmConfig(llmConfig);
                return workflowConfig;
            }
        };
        agentCard.setResourceService(ObjectBuilder.buildResourceService());
        agentCard.setUri("classpath:A2A.json");
        agentCard.setServer("http://xxx");
        agentCard.init();
        AgentCard[] agentCards = agentCard.getAgentCards();
        Assert.assertEquals(2, agentCards.length);
        Assert.assertTrue(agentCards[0].getCapabilities().getStreaming());
        Assert.assertTrue(agentCards[1].getCapabilities().getStreaming());
        Assert.assertEquals("{\"capabilities\":{\"stateTransitionHistory\":false,\"pushNotifications\":true,\"streaming\":true},\"defaultOutputModes\":[\"application/json\",\"image/png\"],\"defaultInputModes\":[\"application/json\",\"text/plain\"],\"preferredTransport\":\"JSONRPC\",\"protocolVersion\":\"0.3.0\",\"description\":\"Provides advanced route planning, traffic analysis, and custom map generation services. This agent can calculate optimal routes, estimate travel times considering real-time traffic, and create personalized maps with points of interest.\",\"skills\":[{\"description\":\"Calculates the optimal driving route between two or more locations, taking into account real-time traffic conditions, road closures, and user preferences (e.g., avoid tolls, prefer highways).\",\"tags\":[\"maps\",\"routing\",\"navigation\",\"directions\",\"traffic\"],\"name\":\"Traffic-Aware Route Optimizer\",\"id\":\"route-optimizer-traffic\"}],\"version\":\"1.2.0\",\"name\":\"video-generator@a2a\",\"url\":\"http://xxx/video-generator@a2a\"}", JsonUtils.write(agentCards[0]));
        Assert.assertEquals("{\"capabilities\":{\"stateTransitionHistory\":true,\"pushNotifications\":true,\"streaming\":true},\"defaultOutputModes\":[\"image/png\"],\"defaultInputModes\":[\"application/json\"],\"preferredTransport\":\"JSONRPC\",\"protocolVersion\":\"0.3.0\",\"description\":\"Stream Provides advanced route planning, traffic analysis, and custom map generation services. This agent can calculate optimal routes, estimate travel times considering real-time traffic, and create personalized maps with points of interest.\",\"skills\":[{\"description\":\"Stream  Calculates the optimal driving route between two or more locations, taking into account real-time traffic conditions, road closures, and user preferences (e.g., avoid tolls, prefer highways).\",\"tags\":[\"maps\",\"routing\"],\"name\":\"Stream  Traffic-Aware Route Optimizer\",\"id\":\"Stream  route-optimizer-traffic\"}],\"version\":\"1.2.0\",\"name\":\"video-generator@a2a_stream\",\"url\":\"http://xxx/video-generator@a2a_stream\"}", JsonUtils.write(agentCards[1]));
        Assert.assertEquals("video-generator@a2a/.well-known/agent-card.json", agentCard.getPatterns()[0]);
        Assert.assertEquals("video-generator@a2a_stream/.well-known/agent-card.json", agentCard.getPatterns()[1]);
    }

    @Test
    public void testUpdateAgentSkill() throws Exception {
        A2ACmdExportAgentCard agentCard = new A2ACmdExportAgentCard() {
            @Override
            protected WorkflowConfig findWorkflowConfig(String[] pair) throws Exception {
                WorkflowConfig workflowConfig = new WorkflowConfig();
                LLMConfig llmConfig = new LLMConfig();
                llmConfig.setStream(true);
                workflowConfig.setLlmConfig(llmConfig);
                return workflowConfig;
            }
        };
        agentCard.setResourceService(ObjectBuilder.buildResourceService());
        agentCard.setUri("classpath:A2A.json");
        agentCard.setServer("http://xxx");
        agentCard.init();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        agentCard.getAgentCards()[0].setSkills(null);
        agentCard.updateAgentSkill(100, agentCard.getAgentCards()[0], workflowConfig);
        Assert.assertNotNull(agentCard.getAgentCards()[0].getSkills().getFirst());
        Assert.assertNotNull(agentCard.getResourceService());
    }

    @Test
    public void testFindWorkflowConfig() throws Exception {
        A2ACmdExportAgentCard agentCard = new A2ACmdExportAgentCard();
        agentCard.setWorkflowConfigService(new WorkflowConfigService() {
            @Override
            public WorkflowConfig config(ConfigSearch configSearch, String workflow) throws Exception {
                Assert.assertEquals(configSearch.getBiz(), "A");
                Assert.assertEquals("B", workflow);
                return null;
            }

            @Override
            public WorkflowConfig config(WorkflowTask workTask, String workflow) throws Exception {
                return null;
            }

            @Override
            public WorkflowConfig config(String biz, String workflow) throws Exception {
                return null;
            }

            @Override
            public WorkflowConfig config(WorkflowTask workTask) throws Exception {
                return null;
            }
        });
        agentCard.findWorkflowConfig(new String[]{"A", "B"});
    }

    @Test
    public void testFindAgentCard1() throws Exception {
        A2ACmdExportAgentCard agentCard = new A2ACmdExportAgentCard() {
            @Override
            protected WorkflowConfig findWorkflowConfig(String[] pair) throws Exception {
                WorkflowConfig workflowConfig = new WorkflowConfig();
                LLMConfig llmConfig = new LLMConfig();
                llmConfig.setStream(true);
                workflowConfig.setLlmConfig(llmConfig);
                return workflowConfig;
            }
        };
        agentCard.setResourceService(ObjectBuilder.buildResourceService());
        agentCard.setUri("classpath:A2A.json");
        agentCard.setServer("http://xxx");
        agentCard.init();
        NettyA2ARequest a2ARequest = NettyA2ARequest.builder()
                .path("video-generator@a2a/.well-known/agent-card.json")
                .build();
        AgentCard card = agentCard.findAgentCard(a2ARequest);
        Assert.assertEquals(card.getName(), "video-generator@a2a");
        Assert.assertTrue(agentCard.support(a2ARequest));
        a2ARequest.setPath("HELLO WORLD");
        Assert.assertFalse(agentCard.support(a2ARequest));
    }

    @Test
    public void testFindAgentCard2() throws Exception {
        A2ACmdExportAgentCard agentCard = new A2ACmdExportAgentCard() {
            @Override
            protected WorkflowConfig findWorkflowConfig(String[] pair) throws Exception {
                WorkflowConfig workflowConfig = new WorkflowConfig();
                LLMConfig llmConfig = new LLMConfig();
                llmConfig.setStream(true);
                workflowConfig.setLlmConfig(llmConfig);
                return workflowConfig;
            }
        };
        agentCard.setResourceService(ObjectBuilder.buildResourceService());
        agentCard.setUri("classpath:A2A.json");
        agentCard.setServer("http://xxx");
        agentCard.init();
        NettyA2ARequest a2ARequest = NettyA2ARequest.builder()
                .path("video-generator@a2a/.well-known/agent-card.json")
                .build();
        a2ARequest.setContent(ImmutableMap.of(NettyA2ARequest.METHOD,"METHOD"));
        Assert.assertFalse(agentCard.support(a2ARequest));
    }

    @Test
    public void testCmd() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn(NettyAttributes.HTTP_CORS).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.replay(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_request.json").openStream(), Map.class))
                .context(context)
                .build();
        A2ACmdExportAgentCard agentCard = new A2ACmdExportAgentCard() {

            @Override
            protected AgentCard findAgentCard(A2ARequest a2aRequest) throws Exception {
                return new AgentCard();
            }

            @Override
            protected WorkflowConfig findWorkflowConfig(String[] pair) throws Exception {
                WorkflowConfig workflowConfig = new WorkflowConfig();
                LLMConfig llmConfig = new LLMConfig();
                llmConfig.setStream(true);
                workflowConfig.setLlmConfig(llmConfig);
                return workflowConfig;
            }
        };
        agentCard.setResourceService(ObjectBuilder.buildResourceService());
        agentCard.setUri("classpath:A2A.json");
        agentCard.setServer("http://xxx");
        agentCard.init();
        agentCard.cmd(a2ARequest);
    }

    @Test
    public void testInit1() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        EasyMock.replay(workflowConfigService);
        A2ACmdExportAgentCard.InitConfig initConfig = new A2ACmdExportAgentCard.InitConfig() {
            @Override
            protected String buildIP() {
                return "127.0.0.1";
            }
        };
        initConfig.setWorkflowConfigService(workflowConfigService);
        initConfig.setServer("SERVER");
        initConfig.setPort(999);
        initConfig.setUri("URI");
        A2ACmdExportAgentCard agentCard = initConfig.a2aCmdExportAgentCard();
        Assert.assertEquals(agentCard.getWorkflowConfigService(), initConfig.getWorkflowConfigService());
        Assert.assertEquals(agentCard.getServer(), initConfig.getServer());
        Assert.assertEquals(agentCard.getUri(), initConfig.getUri());
        Assert.assertEquals(Integer.valueOf(999), initConfig.getPort());
        EasyMock.verify(workflowConfigService);
    }

    @Test
    public void testInit2() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        EasyMock.replay(workflowConfigService);
        A2ACmdExportAgentCard.InitConfig initConfig = new A2ACmdExportAgentCard.InitConfig() {
            @Override
            protected String buildIP() {
                return "127.0.0.1";
            }
        };
        initConfig.setWorkflowConfigService(workflowConfigService);
        initConfig.setPort(999);
        initConfig.setUri("URI");
        A2ACmdExportAgentCard agentCard = initConfig.a2aCmdExportAgentCard();
        Assert.assertEquals(agentCard.getWorkflowConfigService(), initConfig.getWorkflowConfigService());
        Assert.assertTrue(agentCard.getServer().endsWith("999"));
        Assert.assertEquals(agentCard.getUri(), initConfig.getUri());
        Assert.assertEquals(Integer.valueOf(999), initConfig.getPort());
        EasyMock.verify(workflowConfigService);
    }
    @Test
    public void testFindAgentCardEmpty() throws Exception {
        A2ACmdExportAgentCard service = new A2ACmdExportAgentCard();
        service.setPatterns(new String[0]);
        Assert.assertNull(service.findAgentCard(NettyA2ARequest.builder().build()));
    }

    /**
     * 覆盖 InitConfig.buildIP()：return IPUtils.getIP()（沙箱下 getifaddrs 可能失败则跳过）
     */
    @Test
    public void testBuildIP() throws Exception {
        InitConfigForBuildIP initConfig = new InitConfigForBuildIP();
        try {
            String ip = initConfig.callBuildIP();
            Assert.assertNotNull(ip);
        } catch (Exception e) {
            String msg = e.getMessage();
            if (msg == null && e.getCause() != null) {
                msg = e.getCause().getMessage();
            }
            if (msg != null && msg.contains("getifaddrs")) {
                org.junit.Assume.assumeTrue("getifaddrs not permitted in sandbox", false);
            }
            throw e;
        }
    }

    private static class InitConfigForBuildIP extends A2ACmdExportAgentCard.InitConfig {
        public String callBuildIP() throws Exception {
            return buildIP();
        }
    }
}
