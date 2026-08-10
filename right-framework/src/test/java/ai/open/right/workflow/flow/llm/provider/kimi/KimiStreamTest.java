package ai.open.right.workflow.flow.llm.provider.kimi;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStream;
import ai.open.right.workflow.flow.llm.signal.SignalExecutor;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import com.fasterxml.jackson.databind.exc.MismatchedInputException;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Set;

import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
public class KimiStreamTest {

    @Test
    public void testOnceWithFinished() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        h.store(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyString(), EasyMock.anyObject(), EasyMock.anyInt(), EasyMock.anyInt(), EasyMock.anyLong());
        EasyMock.expectLastCall().anyTimes();
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        EasyMock.expect(c.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(c.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getStoreCompleted()).andReturn(true).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(c.getMessage(), c.getRepositories(), mockHistories, c.getExpired(), c.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(new TokenStatistic() {

            @Override
            public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
                Assert.assertEquals(Integer.valueOf(1991), tokenData.getTotal());
                Assert.assertEquals(Integer.valueOf(0), tokenData.getCache());
            }

            @Override
            public List<TokenData> readAll(Dimension dimension, List<String> model) throws Exception {
                return List.of();
            }

            @Override
            public List<TokenData> readAll(Dimension dimension) throws Exception {
                return List.of();
            }

            @Override
            public TokenData read(Dimension dimension, String model) throws Exception {
                return null;
            }

            @Override
            public TokenData read(Dimension dimension) throws Exception {
                return null;
            }

            @Override
            public Set<String> models() throws Exception {
                return Set.of();
            }
        })
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {

            @Override
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals("UNKNOWN", historyPairs.getFirst().getQuery());
                Assert.assertEquals(Long.valueOf(c.getMessage().getCreated() + 1), Long.valueOf(historyPairs.getFirst().getCreated()));
                Assert.assertTrue(historyPairs.getLast().getAnswer().contains("Let's first"));
                Assert.assertTrue(Long.valueOf(c.getMessage().getCreated()) <= Long.valueOf(historyPairs.getLast().getCreated()));
                return mockHistories;
            }
        };
        Assert.assertTrue(stream.atonce("{\"id\":\"chatcmpl-22b53918dadd425aa6565edb59154607\",\"object\":\"chat.completion\",\"created\":1723457352,\"model\":\"moonshot-v1-8k\",\"choices\":[{\"index\":0,\"message\":{\"role\":\"assistant\",\"content\":\"### Let's first understand the problem and devise a plan to solve the problem.\\n\\n作为一位资深的网络销售专家，我的任务是帮助新用户了解并选购我们公司的商品。首先，我需要识别用户的意图是想了解商品还是想购买商品。如果用户想了解商品，我会根据用户的兴趣或者随机挑选商品进行介绍；如果用户想购买商品，我会提供购买链接。在整个过程中，我会使用我的专业知识和沟通技巧，确保用户得到满意的服务。\\n\\n### Then, let's carry out the plan and solve the problem step by step.\\n\\n1. **意图识别**：我需要了解用户是想了解商品还是想购买商品。如果用户没有明确表达意图，我会使用介绍公司的技巧，同时鼓励用户提供更多信息。\\n\\n2. **商品介绍**：如果用户想了解商品，我会根据用户的兴趣或者随机挑选商品进行介绍，包括商品的价格、规格和卖点。\\n\\n3. **购买链接提供**：如果用户已经决定购买，我会提供一个直接的购买链接。\\n\\n现在，让我们开始与用户的互动，根据他们的需求提供帮助。\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1770,\"completion_tokens\":221,\"total_tokens\":1991}}"));
        EasyMock.verify(s, t, h, c);
    }

    @Test(expected = MismatchedInputException.class)
    public void testOnceWithFinishedWithJsonError() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        h.store(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyString(), EasyMock.anyObject(), EasyMock.anyInt(), EasyMock.anyInt(), EasyMock.anyLong());
        EasyMock.expectLastCall().anyTimes();
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build());
        try {
            stream.atonce("\"id\":\"chatcmpl-22b53918dadd425aa6565edb59154607\",\"object\":\"chat.completion\",\"created\":1723457352,\"model\":\"moonshot-v1-8k\",\"choices\":[{\"index\":0,\"message\":{\"role\":\"assistant\",\"content\":\"### Let's first understand the problem and devise a plan to solve the problem.\\n\\n作为一位资深的网络销售专家，我的任务是帮助新用户了解并选购我们公司的商品。首先，我需要识别用户的意图是想了解商品还是想购买商品。如果用户想了解商品，我会根据用户的兴趣或者随机挑选商品进行介绍；如果用户想购买商品，我会提供购买链接。在整个过程中，我会使用我的专业知识和沟通技巧，确保用户得到满意的服务。\\n\\n### Then, let's carry out the plan and solve the problem step by step.\\n\\n1. **意图识别**：我需要了解用户是想了解商品还是想购买商品。如果用户没有明确表达意图，我会使用介绍公司的技巧，同时鼓励用户提供更多信息。\\n\\n2. **商品介绍**：如果用户想了解商品，我会根据用户的兴趣或者随机挑选商品进行介绍，包括商品的价格、规格和卖点。\\n\\n3. **购买链接提供**：如果用户已经决定购买，我会提供一个直接的购买链接。\\n\\n现在，让我们开始与用户的互动，根据他们的需求提供帮助。\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1770,\"completion_tokens\":221,\"total_tokens\":1991}}");
            Assert.fail();
        } finally {
            EasyMock.verify(s, t, h, c);
        }
    }

    @Test(expected = MismatchedInputException.class)
    public void testOnceWithFinishedWithNull() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        h.store(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyString(), EasyMock.anyObject(), EasyMock.anyInt(), EasyMock.anyInt(), EasyMock.anyLong());
        EasyMock.expectLastCall().anyTimes();
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build());
        try {
            stream.atonce("\"id\":\"chatcmpl-22b53918dadd425aa6565edb59154607\",\"object\":\"chat.completion\",\"created\":1723457352,\"model\":\"moonshot-v1-8k\",\"choices\":[{\"index\":0,\"message\":{\"role\":\"assistant\",\"content\":\"### Let's first understand the problem and devise a plan to solve the problem.\\n\\n作为一位资深的网络销售专家，我的任务是帮助新用户了解并选购我们公司的商品。首先，我需要识别用户的意图是想了解商品还是想购买商品。如果用户想了解商品，我会根据用户的兴趣或者随机挑选商品进行介绍；如果用户想购买商品，我会提供购买链接。在整个过程中，我会使用我的专业知识和沟通技巧，确保用户得到满意的服务。\\n\\n### Then, let's carry out the plan and solve the problem step by step.\\n\\n1. **意图识别**：我需要了解用户是想了解商品还是想购买商品。如果用户没有明确表达意图，我会使用介绍公司的技巧，同时鼓励用户提供更多信息。\\n\\n2. **商品介绍**：如果用户想了解商品，我会根据用户的兴趣或者随机挑选商品进行介绍，包括商品的价格、规格和卖点。\\n\\n3. **购买链接提供**：如果用户已经决定购买，我会提供一个直接的购买链接。\\n\\n现在，让我们开始与用户的互动，根据他们的需求提供帮助。\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1770,\"completion_tokens\":221,\"total_tokens\":1991}}");
            Assert.fail();
        } finally {
            EasyMock.verify(s, t, h, c);
        }
    }

    @Test(expected = IllegalArgumentException.class)
    public void testOnceWithFinishedWithEmptyMessage() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        h.store(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyString(), EasyMock.anyObject(), EasyMock.anyInt(), EasyMock.anyInt(), EasyMock.anyLong());
        EasyMock.expectLastCall().anyTimes();
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build());
        try {
            stream.atonce("{\"id\":\"chatcmpl-22b53918dadd425aa6565edb59154607\",\"object\":\"chat.completion\",\"created\":1723457352,\"model\":\"moonshot-v1-8k\",\"choices_\":[{\"index\":0,\"message_\":{\"role\":\"assistant\",\"content\":\"### Let's first understand the problem and devise a plan to solve the problem.\\n\\n作为一位资深的网络销售专家，我的任务是帮助新用户了解并选购我们公司的商品。首先，我需要识别用户的意图是想了解商品还是想购买商品。如果用户想了解商品，我会根据用户的兴趣或者随机挑选商品进行介绍；如果用户想购买商品，我会提供购买链接。在整个过程中，我会使用我的专业知识和沟通技巧，确保用户得到满意的服务。\\n\\n### Then, let's carry out the plan and solve the problem step by step.\\n\\n1. **意图识别**：我需要了解用户是想了解商品还是想购买商品。如果用户没有明确表达意图，我会使用介绍公司的技巧，同时鼓励用户提供更多信息。\\n\\n2. **商品介绍**：如果用户想了解商品，我会根据用户的兴趣或者随机挑选商品进行介绍，包括商品的价格、规格和卖点。\\n\\n3. **购买链接提供**：如果用户已经决定购买，我会提供一个直接的购买链接。\\n\\n现在，让我们开始与用户的互动，根据他们的需求提供帮助。\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1770,\"completion_tokens\":221,\"total_tokens\":1991}}");
            Assert.fail();
        } finally {
            EasyMock.verify(s, t, h, c);
        }
    }

    @Test
    public void testStreamWithNotFinished() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        h.store(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyString(), EasyMock.anyObject(), EasyMock.anyInt(), EasyMock.anyInt(), EasyMock.anyLong());
        EasyMock.expectLastCall().anyTimes();
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }
        };
        Assert.assertFalse(stream.stream("data: {\"id\":\"chatcmpl-338aa5ca1d0648d48bc0969f44a978ff\",\"object\":\"chat.completion.chunk\",\"created\":1723452974,\"model\":\"moonshot-v1-8k\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"null\",\"usage\":{\"prompt_tokens\":89,\"completion_tokens\":37,\"total_tokens\":126}}]}"));
        EasyMock.verify(s, t, h, c);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testStreamWithFinishedAndWithoutId() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        h.store(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyString(), EasyMock.anyObject(), EasyMock.anyInt(), EasyMock.anyInt(), EasyMock.anyLong());
        EasyMock.expectLastCall().anyTimes();
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }
        };
        try {
            stream.stream("data: {\"id_\":\"chatcmpl-338aa5ca1d0648d48bc0969f44a978ff\",\"object\":\"chat.completion.chunk\",\"created\":1723452974,\"model\":\"moonshot-v1-8k\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"null\",\"usage\":{\"prompt_tokens\":89,\"completion_tokens\":37,\"total_tokens\":126}}]}");
            Assert.fail();
        } finally {
            EasyMock.verify(s, t, h, c);
        }
    }

    @Test(expected = IllegalArgumentException.class)
    public void testStreamWithInValidBody() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        h.store(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyString(), EasyMock.anyObject(), EasyMock.anyInt(), EasyMock.anyInt(), EasyMock.anyLong());
        EasyMock.expectLastCall().anyTimes();
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }
        };
        try {
            stream.stream("{\"id\":\"chatcmpl-338aa5ca1d0648d48bc0969f44a978ff\",\"object\":\"chat.completion.chunk\",\"created\":1723452974,\"model\":\"moonshot-v1-8k\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"null\",\"usage\":{\"prompt_tokens\":89,\"completion_tokens\":37,\"total_tokens\":126}}]}");
            Assert.fail();
        } finally {
            EasyMock.verify(s, t, h, c);
        }
    }

    @Test(expected = MismatchedInputException.class)
    public void testStreamWithInValidJson() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        h.store(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyString(), EasyMock.anyObject(), EasyMock.anyInt(), EasyMock.anyInt(), EasyMock.anyLong());
        EasyMock.expectLastCall().anyTimes();
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }
        };
        try {
            stream.stream("data: \"id\":\"chatcmpl-338aa5ca1d0648d48bc0969f44a978ff\",\"object\":\"chat.completion.chunk\",\"created\":1723452974,\"model\":\"moonshot-v1-8k\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"null\",\"usage\":{\"prompt_tokens\":89,\"completion_tokens\":37,\"total_tokens\":126}}]}");
            Assert.fail();
        } finally {
            EasyMock.verify(s, t, h, c);
        }
    }


    @Test
    public void testStreamWithFinished() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn(null).anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("SUFFIX").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(new TokenStatistic() {

            @Override
            public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
                Assert.assertEquals(Integer.valueOf(126), tokenData.getTotal());
                Assert.assertEquals(Integer.valueOf(0), tokenData.getCache());
            }

            @Override
            public List<TokenData> readAll(Dimension dimension, List<String> model) throws Exception {
                return List.of();
            }

            @Override
            public List<TokenData> readAll(Dimension dimension) throws Exception {
                return List.of();
            }

            @Override
            public TokenData read(Dimension dimension, String model) throws Exception {
                return null;
            }

            @Override
            public TokenData read(Dimension dimension) throws Exception {
                return null;
            }

            @Override
            public Set<String> models() throws Exception {
                return Set.of();
            }
        })
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {

            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }

            @Override
            protected void storeConversation(String content) throws Exception {
            }
        };
        Assert.assertTrue(stream.stream("data: {\"id\":\"chatcmpl-338aa5ca1d0648d48bc0969f44a978ff\",\"object\":\"chat.completion.chunk\",\"created\":1723452974,\"model\":\"moonshot-v1-8k\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"HELLO\"},\"finish_reason\":\"stop\",\"usage\":{\"prompt_tokens\":89,\"completion_tokens\":37,\"total_tokens\":126}}]}"));
        EasyMock.verify(s, h, c, t);
    }
}
