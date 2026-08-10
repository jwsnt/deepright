package ai.open.right.workflow.flow.llm.provider.deepseek;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderStream;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStreamFunCall;
import ai.open.right.workflow.flow.llm.signal.SignalExecutor;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import com.fasterxml.jackson.databind.exc.MismatchedInputException;
import com.google.common.collect.ImmutableMap;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.reflect.MethodUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.lang.reflect.Method;
import java.nio.charset.Charset;
import java.nio.charset.StandardCharsets;
import java.util.*;

import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
public class DeepSeekStreamTest {

    @Test
    public void testOnceWithFinished() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = newStreamTestOpenAiRequestWorkflow();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void storeConversation(String content) throws Exception {

            }
        };
        Assert.assertTrue(stream.atonce("{\"id\":\"chatcmpl-22b53918dadd425aa6565edb59154607\",\"object\":\"chat.completion\",\"created\":1723457352,\"model\":\"moonshot-v1-8k\",\"choices\":[{\"index\":0,\"message\":{\"role\":\"assistant\",\"content\":\"### Let's first understand the problem and devise a plan to solve the problem.\\n\\n作为一位资深的网络销售专家，我的任务是帮助新用户了解并选购我们公司的商品。首先，我需要识别用户的意图是想了解商品还是想购买商品。如果用户想了解商品，我会根据用户的兴趣或者随机挑选商品进行介绍；如果用户想购买商品，我会提供购买链接。在整个过程中，我会使用我的专业知识和沟通技巧，确保用户得到满意的服务。\\n\\n### Then, let's carry out the plan and solve the problem step by step.\\n\\n1. **意图识别**：我需要了解用户是想了解商品还是想购买商品。如果用户没有明确表达意图，我会使用介绍公司的技巧，同时鼓励用户提供更多信息。\\n\\n2. **商品介绍**：如果用户想了解商品，我会根据用户的兴趣或者随机挑选商品进行介绍，包括商品的价格、规格和卖点。\\n\\n3. **购买链接提供**：如果用户已经决定购买，我会提供一个直接的购买链接。\\n\\n现在，让我们开始与用户的互动，根据他们的需求提供帮助。\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1770,\"completion_tokens\":221,\"total_tokens\":1991}}"));
        EasyMock.verify(s, t, h);
    }

    @Test(expected = MismatchedInputException.class)
    public void testOnceWithFinishedWithJsonError() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        OpenAiRequest c = newStreamTestOpenAiRequest();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            EasyMock.verify(s, t, h);
        }
    }

    @Test(expected = MismatchedInputException.class)
    public void testOnceWithFinishedWithNull() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        OpenAiRequest c = newStreamTestOpenAiRequest();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            EasyMock.verify(s, t, h);
        }
    }

    @Test(expected = IllegalArgumentException.class)
    public void testOnceWithFinishedWithEmptyMessage() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        OpenAiRequest c = newStreamTestOpenAiRequest();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            EasyMock.verify(s, t, h);
        }
    }

    @Test
    public void testStreamWithNotFinished() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        OpenAiRequest c = newStreamTestOpenAiRequest();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
        };
        Assert.assertFalse(stream.stream("data: {\"id\":\"chatcmpl-338aa5ca1d0648d48bc0969f44a978ff\",\"object\":\"chat.completion.chunk\",\"created\":1723452974,\"model\":\"moonshot-v1-8k\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"null\",\"usage\":{\"prompt_tokens\":89,\"completion_tokens\":37,\"total_tokens\":126}}]}"));
        EasyMock.verify(s, t, h);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testStreamWithFinishedAndWithoutId() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        OpenAiRequest c = newStreamTestOpenAiRequest();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            EasyMock.verify(s, t, h);
        }
    }

    @Test(expected = IllegalArgumentException.class)
    public void testStreamWithInValidBody() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        OpenAiRequest c = newStreamTestOpenAiRequest();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            EasyMock.verify(s, t, h);
        }
    }

    @Test(expected = MismatchedInputException.class)
    public void testStreamWithInValidJson() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        OpenAiRequest c = newStreamTestOpenAiRequest();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            EasyMock.verify(s, t, h);
        }
    }


    @Test
    public void testStreamWithFinished() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = newStreamTestOpenAiRequestWorkflow();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
        EasyMock.verify(s, h, t);
    }

    @Test
    public void testStreamWithFunCall1() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO");
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = newStreamTestOpenAiRequestFunCall();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void notifySegment() throws Exception {

            }

            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }

            @Override
            protected void storeConversation() throws Exception {
            }
        };
        for (int i = 1; i <= 17; i++) {
            String filename = "DeepSeekResponse_funcall_s" + i + ".json";
            String request = IOUtils.toString(ResourceUtils.getURL("classpath:" + filename).openStream(), StandardCharsets.UTF_8);
            if (i != 17) {
                Assert.assertFalse(stream.stream("data: " + request));
            } else {
                Assert.assertTrue(stream.stream("data: " + request));
            }
        }
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(stream.getProviderFunRequests().size()));
        String expect = "{\"index\":0,\"id\":\"call_711f0d9131be4ed9a84be676\",\"type\":\"function\",\"function\":{\"arguments\":\"{\\\"filename\\\": \\\"/Users/shenjiawei/DEV/bff/JumpLink.service.ts\\\", \\\"why_do_this\\\": \\\"检查JumpLink.service.ts文件是否存在，确保后续操作的基础条件\\\"}\",\"name\":\"Tools_shell__check_file_exist\"}}";
        Assert.assertEquals(expect, JsonUtils.write(stream.getProviderFunRequests().getFirst().getRefer()));
        Assert.assertFalse(JsonUtils.read(JsonUtils.write(stream.getProviderFunRequests().getFirst().getRefer()), Map.class).isEmpty());
        Assert.assertEquals("{\"filename\": \"/Users/shenjiawei/DEV/bff/JumpLink.service.ts\", \"why_do_this\": \"检查JumpLink.service.ts文件是否存在，确保后续操作的基础条件\"}", stream.getProviderFunRequests().getFirst().getArgs());
        Assert.assertFalse(JsonUtils.read(JsonUtils.write(stream.getProviderFunRequests().getFirst().getArgs()), Map.class).isEmpty());
        Assert.assertEquals("Tools_shell__check_file_exist", stream.getProviderFunRequests().getFirst().getName());
        Assert.assertEquals("call_711f0d9131be4ed9a84be676", stream.getProviderFunRequests().getFirst().getId());
        EasyMock.verify(s, h, t);
    }

    @Test
    public void testStreamWithFunCall2() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO");
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = newStreamTestOpenAiRequestFunCall();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void notifySegment() throws Exception {
                Assert.assertEquals("HELLO WORLDHELLO", this.segment.getContent().toString());
                Assert.assertTrue(this.segment.isFinished());
            }

            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }

            @Override
            protected void storeConversation() throws Exception {
            }
        };
        for (int i = 0; i <= 18; i++) {
            String filename = "DeepSeekResponse_funcall_s1_" + i + ".json";
            String request = IOUtils.toString(ResourceUtils.getURL("classpath:" + filename).openStream(), StandardCharsets.UTF_8);
            if (i != 18) {
                Assert.assertFalse(stream.stream("data: " + request));
            } else {
                Assert.assertTrue(stream.stream("data: " + request));
            }
        }
        Assert.assertEquals(Integer.valueOf(2), Integer.valueOf(stream.getProviderFunRequests().size()));
        String expect = "{\"index\":0,\"id\":\"call_711f0d9131be4ed9a84be676\",\"type\":\"function\",\"function\":{\"arguments\":\"{\\\"filename\\\": \\\"/Users/shenjiawei/DEV/bff/JumpLink.service.ts\\\", \\\"why_do_this\\\": \\\"检查JumpLink.service.ts文件是否存在，确保后续操作的基础条件\\\"}\",\"name\":\"Tools_shell__check_file_exist\"}}";
        Assert.assertEquals(expect, JsonUtils.write(stream.getProviderFunRequests().getFirst().getRefer()));
        Assert.assertFalse(JsonUtils.read(JsonUtils.write(stream.getProviderFunRequests().getFirst().getRefer()), Map.class).isEmpty());
        Assert.assertEquals("{\"filename\": \"/Users/shenjiawei/DEV/bff/JumpLink.service.ts\", \"why_do_this\": \"检查JumpLink.service.ts文件是否存在，确保后续操作的基础条件\"}", stream.getProviderFunRequests().getFirst().getArgs());
        Assert.assertFalse(JsonUtils.read(JsonUtils.write(stream.getProviderFunRequests().getFirst().getArgs()), Map.class).isEmpty());
        Assert.assertEquals("Tools_shell__check_file_exist", stream.getProviderFunRequests().getFirst().getName());
        Assert.assertEquals("call_711f0d9131be4ed9a84be676", stream.getProviderFunRequests().getFirst().getId());
        Assert.assertEquals("{\"index\":1,\"id\":\"HELLO_WORLD\",\"type\":\"function\",\"function\":{\"name\":\"TOOLS_MY_TOOLS\",\"arguments\":\"{\\\"filename\\\": \\\"/Users2\\\"}\"}}", JsonUtils.write(stream.getProviderFunRequests().getLast().getRefer()));
        Assert.assertFalse(JsonUtils.read(JsonUtils.write(stream.getProviderFunRequests().getLast().getRefer()), Map.class).isEmpty());
        Assert.assertEquals("{\"filename\": \"/Users/shenjiawei/DEV/bff/JumpLink.service.ts\", \"why_do_this\": \"检查JumpLink.service.ts文件是否存在，确保后续操作的基础条件\"}", stream.getProviderFunRequests().getFirst().getArgs());
        Assert.assertFalse(JsonUtils.read(JsonUtils.write(stream.getProviderFunRequests().getLast().getArgs()), Map.class).isEmpty());
        Assert.assertEquals("TOOLS_MY_TOOLS", stream.getProviderFunRequests().getLast().getName());
        Assert.assertEquals("HELLO_WORLD", stream.getProviderFunRequests().getLast().getId());
        Assert.assertEquals("HELLO WORLDHELLO", stream.getContentBuffer().toString());
        EasyMock.verify(s, h, t);
    }

    @Test
    public void testAddReason() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO");
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = newStreamTestOpenAiRequestReasoningStore(true);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void notifySegment() throws Exception {
                Assert.assertEquals("HELLO WORLDHELLO", this.segment.getContent().toString());
                Assert.assertTrue(this.segment.isFinished());
            }

            @Override
            protected void notify(int seqid, boolean finished) throws Exception {

            }

            @Override
            protected void storeConversation() throws Exception {
            }
        };
        Assert.assertNull(stream.getReasoning());
        invokeAddReason(stream, ImmutableMap.of("reasoning_content", "A"), false);
        Assert.assertEquals("A", stream.getReasoning().toString());
        invokeAddReason(stream, ImmutableMap.of("reasoning_content", "B"), false);
        Assert.assertEquals("AB", stream.getReasoning().toString());
        Assert.assertEquals("AB", stream.getContentBuffer().toString());
        EasyMock.verify(s, h, t);
    }

    @Test
    public void testAfterCreate() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO");
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = newStreamTestOpenAiRequestReasoningStore(true);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void notifySegment() throws Exception {
                Assert.assertEquals("HELLO WORLDHELLO", this.segment.getContent().toString());
                Assert.assertTrue(this.segment.isFinished());
            }

            @Override
            protected void notify(int seqid, boolean finished) throws Exception {

            }

            @Override
            protected void storeConversation() throws Exception {
            }
        };
        Assert.assertNull(stream.getReasoning());
        invokeAddReason(stream, ImmutableMap.of("reasoning_content", "A"), false);
        Assert.assertEquals("A", stream.getReasoning().toString());
        ProviderFunCallRequest providerFunCallRequest = ProviderFunCallRequest.builder().build();
        invokeAfterCreateFunRequest(stream, providerFunCallRequest, ImmutableMap.of(), ImmutableMap.of(), "A", "B", "C");
        Assert.assertEquals("A", providerFunCallRequest.getReason());
        invokeAddReason(stream, ImmutableMap.of("reasoning_content", "B"), false);
        Assert.assertEquals("A", providerFunCallRequest.getReason());
        EasyMock.verify(s, h, t);
    }

    @Test
    public void testAfterUpdate() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO");
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = newStreamTestOpenAiRequestReasoningStore(true);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void notifySegment() throws Exception {
                Assert.assertEquals("HELLO WORLDHELLO", this.segment.getContent().toString());
                Assert.assertTrue(this.segment.isFinished());
            }

            @Override
            protected void notify(int seqid, boolean finished) throws Exception {

            }

            @Override
            protected void storeConversation() throws Exception {
            }
        };
        Assert.assertNull(stream.getReasoning());
        invokeAddReason(stream, ImmutableMap.of("reasoning_content", "A"), false);
        Assert.assertEquals("A", stream.getReasoning().toString());
        ProviderFunCallRequest providerFunCallRequest = ProviderFunCallRequest.builder().build();
        invokeAfterUpdateFunRequest(stream, providerFunCallRequest, ImmutableMap.of(), ImmutableMap.of(), "A", "B", "C");
        Assert.assertEquals("A", providerFunCallRequest.getReason());
        invokeAddReason(stream, ImmutableMap.of("reasoning_content", "B"), false);
        Assert.assertEquals("A", providerFunCallRequest.getReason());
        EasyMock.verify(s, h, t);
    }

    @Test
    public void testStoreConversation() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO");
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = newStreamTestOpenAiRequestReasoningStore(true);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(c.getMessage(), c.getRepositories(), mockHistories, c.getExpired(), c.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void notifySegment() throws Exception {
                Assert.assertEquals("HELLO WORLD HELLO", this.segment.getContent().toString());
                Assert.assertTrue(this.segment.isFinished());
            }

            @Override
            protected void notify(int seqid, boolean finished) throws Exception {

            }

            @Override
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals("UNKNOWN", historyPairs.getFirst().getQuery());
                Assert.assertEquals(Long.valueOf(c.getMessage().getCreated() + 1), Long.valueOf(historyPairs.getFirst().getCreated()));
                Assert.assertTrue(historyPairs.getLast().getAnswer().contains("CONTENT"));
                Assert.assertTrue(Long.valueOf(c.getMessage().getCreated()) <= Long.valueOf(historyPairs.getLast().getCreated()));
                return mockHistories;
            }
        };
        Assert.assertNull(stream.getReasoning());
        invokeAddReason(stream, ImmutableMap.of("reasoning_content", "A"), false);
        Method method = MethodUtils.getMatchingMethod(ProviderStream.class, "storeConversation", String.class);
        method.setAccessible(true);
        method.invoke(stream, "CONTENT");
        EasyMock.verify(s, h, t);
    }

    @Test
    public void testAddReason3() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        OpenAiRequest c = newStreamTestOpenAiRequestWorkflow();
        c.setPrintReason(true);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void addReason(Map<String, Object> message, Boolean finished) throws Exception {
                if (Boolean.TRUE.equals(finished)) {
                    Assert.assertNull(message);
                    return;
                }
                Assert.assertEquals("首先，用户的指令是让我检查提交的代码有没有质量问题。我是代码自动化CR助理，要求500字，在结尾处输出`[FINISH]`。需要先输出有问题的代码，然后输出需要修改的问题，不回答无关内容。\n" +
                        "\n" +
                        "代码看起来是一个Spring Boot的MainApplication类。我需要检查它是否有质量问题。\n" +
                        "\n" +
                        "让我分析代码：\n" +
                        "\n" +
                        "1. **导入语句**：导入了必要的Spring Boot相关类，看起来标准。\n" +
                        "\n" +
                        "2. **注解**：\n" +
                        "   - `@PropertySource(\"classpath:application.properties\")`：指定属性源。\n" +
                        "   - `@EnableAsync(proxyTargetClass = true)`：启用异步，并设置代理目标类。\n" +
                        "   - `@SpringBootApplication(exclude = {DataSourceAutoConfiguration.class})`：Spring Boot应用主类，排除了数据源自动配置。\n" +
                        "\n" +
                        "3. **类定义**：`public class MainApplication`\n" +
                        "\n" +
                        "4. **main方法**：`public static void main(String[] args) throws Exception`，调用了`SpringApplication.run(MainApplication.class, args)`。\n" +
                        "\n" +
                        "现在，检查潜在的质量问题：\n" +
                        "\n" +
                        "- **异常处理**：main方法抛出了`Exception`。在Spring Boot应用中，main方法通常不抛出异常，或者SpringApplication.run会处理异常。但这里抛出了Exception，这可能是不必要的，因为SpringApplication.run已经捕获了异常。如果真的有异常，应用会停止，但抛出Exception在main方法中可能不是最佳实践。\n" +
                        "\n" +
                        "- **属性源**：使用`@PropertySource(\"classpath:application.properties\")`，但Spring Boot默认会加载classpath下的application.properties。所以这个注解可能是多余的，除非有特殊需求。\n" +
                        "\n" +
                        "- **排除数据源配置**：`@SpringBootApplication(exclude = {DataSourceAutoConfiguration.class})`，排除数据源自动配置。这可能是故意的，如果应用不使用数据库。但需要确认是否有这样的需求。如果没有明确说明，可能是一个问题，因为通常Spring Boot应用可能会使用数据库。\n" +
                        "\n" +
                        "- **包名**：包名是`ai.open.right`，看起来没问题。\n" +
                        "\n" +
                        "- **代码风格**：结构简洁，符合Spring Boot标准。\n" +
                        "\n" +
                        "- **其他质量问题**：\n" +
                        "  - 缺少注释或文档。\n" +
                        "  - 没有处理命令行参数或额外配置。\n" +
                        "  - 代理目标类设置：`proxyTargetClass = true` 可能不是必需的，取决于是否需要CGLIB代理。\n" +
                        "\n" +
                        "列出有问题的部分：\n" +
                        "\n" +
                        "1. **多余的属性源注解**：Spring Boot默认加载application.properties，所以`@PropertySource`可能不需要。\n" +
                        "\n" +
                        "2. **main方法抛出Exception**：不必要地抛出异常。\n" +
                        "\n" +
                        "3. **排除数据源配置**：如果没有明确理由，排除数据源可能是个问题，但考虑到这是用户提供的代码，可能是有意的。所以需要谨慎。\n" +
                        "\n" +
                        "用户说“检查提交的代码有没有质量问题”，所以我应该基于一般最佳实践来检查。\n" +
                        "\n" +
                        "最终输出格式：先输出有问题的代码，然后输出需要修改的问题。\n" +
                        "\n" +
                        "有问题的代码：我应该引用具体的代码行。\n" +
                        "\n" +
                        "例如：\n" +
                        "- 第1行：package语句没问题。\n" +
                        "- 导入语句没问题。\n" +
                        "- 注解部分：第8行的`@PropertySource`和第11行的`@SpringBootApplication`排除。\n" +
                        "\n" +
                        "输出时，可以复制有问题的代码片段。\n" +
                        "\n" +
                        "字数要求500字，但问题可能不多，所以我可以详细解释。\n" +
                        "\n" +
                        "构建响应：\n" +
                        "\n" +
                        "1. **输出有问题的代码**：\n" +
                        "   - `@PropertySource(\"classpath:application.properties\")`\n" +
                        "   - `public static void main(String[] args) throws Exception`\n" +
                        "\n" +
                        "2. **输出需要修改的问题**：\n" +
                        "   - 问题1：`@PropertySource`注解是多余的，因为Spring Boot自动加载application.properties。建议移除。\n" +
                        "   - 问题2：main方法抛出Exception是不必要的，因为SpringApplication.run内部处理异常。建议改为`public static void main(String[] args)`。\n" +
                        "   - 问题3：`@SpringBootApplication`排除了`DataSourceAutoConfiguration.class`。如果应用不使用数据库，这是合理的，但需要确认。否则，可能需要考虑是否需要数据源配置。\n" +
                        "\n" +
                        "其他可能的改进：添加注释或文档。\n" +
                        "\n" +
                        "确保在结尾输出`[FINISH]`。\n" +
                        "\n" +
                        "现在，写响应：\n" +
                        "\n" +
                        "首先，输出有问题的代码：\n" +
                        "\n" +
                        "```\n" +
                        "@PropertySource(\"classpath:application.properties\")\n" +
                        "public static void main(String[] args) throws Exception\n" +
                        "```\n" +
                        "\n" +
                        "然后，输出修改的问题：\n" +
                        "\n" +
                        "1. 移除`@PropertySource`注解，因为Spring Boot默认加载classpath下的application.properties文件，添加此注解是冗余的。\n" +
                        "\n" +
                        "2. 修改main方法的签名，移除`throws Exception`。SpringApplication.run方法已经处理了异常，不需要在main方法中抛出。\n" +
                        "\n" +
                        "3. 检查是否需要排除DataSourceAutoConfiguration。如果应用不使用数据库，排除是合理的；否则，应考虑移除排除或添加数据源配置。\n" +
                        "\n" +
                        "最后，输出`[FINISH]`。\n" +
                        "\n" +
                        "控制字数在500字左右。既然问题不多，我可以扩展解释。", message.get("reasoning_content"));
            }

            @Override
            protected void storeConversation(String content) throws Exception {
            }
        };
        Assert.assertTrue(stream.atonce(IOUtils.toString(ResourceUtils.getURL("classpath:DeepSeekResponse_Reason_Content.json").openStream(), Charset.defaultCharset())));
        EasyMock.verify(s, t);
    }

    @Test
    public void testStoreCompleted() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO");
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = newStreamTestOpenAiRequestReasoningStore(false);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(c.getMessage(), c.getRepositories(), mockHistories, c.getExpired(), c.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void notifySegment() throws Exception {
                Assert.assertEquals("HELLO WORLD HELLO", this.segment.getContent().toString());
                Assert.assertTrue(this.segment.isFinished());
            }

            @Override
            protected void notify(int seqid, boolean finished) throws Exception {

            }

            @Override
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(historyPairs.size()));
                Assert.assertTrue(historyPairs.getFirst().getAnswer().contains("CONTENT"));
                Assert.assertTrue(Long.valueOf(c.getMessage().getCreated()) <= Long.valueOf(historyPairs.getFirst().getCreated()));
                return mockHistories;
            }
        };
        Assert.assertNull(stream.getReasoning());
        invokeAddReason(stream, ImmutableMap.of("reasoning_content", "A"), false);
        Method method = MethodUtils.getMatchingMethod(ProviderStream.class, "storeConversation", String.class);
        method.setAccessible(true);
        method.invoke(stream, "CONTENT");
        EasyMock.verify(s, h, t);
    }

    private static OpenAiRequest newStreamTestOpenAiRequest() {
        OpenAiRequest c = new OpenAiRequest();
        c.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        c.setPureQuery(false);
        c.setRepositories(new ArrayList<>(Arrays.asList("UNKNOWN")));
        c.setChain("NEXT_WORKFLOW");
        c.setTokenFirst(1024);
        c.setTokenBuffer(1024);
        c.setStream(false);
        c.setContainHistories(true);
        c.setHistories(null);
        c.setPrefix("");
        c.setSuffix("");
        c.setNotifier(null);
        c.setStoreFunCall(false);
        c.setMetadata(null);
        return c;
    }

    private static OpenAiRequest newStreamTestOpenAiRequestWorkflow() {
        OpenAiRequest c = newStreamTestOpenAiRequest();
        c.setScene("WORKFLOW");
        c.setExpired(1000);
        c.setWriteable(true);
        return c;
    }

    private static OpenAiRequest newStreamTestOpenAiRequestFunCall() {
        OpenAiRequest c = newStreamTestOpenAiRequestWorkflow();
        c.setFunCallTimeout(1000);
        return c;
    }

    private static OpenAiRequest newStreamTestOpenAiRequestReasoningStore(boolean storeCompleted) {
        OpenAiRequest c = newStreamTestOpenAiRequestFunCall();
        c.setPrintReason(true);
        c.setStoreCompleted(storeCompleted);
        return c;
    }

    private static void invokeAddReason(OpenAiStreamFunCall stream, Map<String, Object> message, boolean finished) throws Exception {
        Method m = MethodUtils.getMatchingMethod(OpenAiStreamFunCall.class, "addReason", Map.class, Boolean.class);
        m.setAccessible(true);
        m.invoke(stream, message, finished);
    }

    private static void invokeAfterCreateFunRequest(OpenAiStreamFunCall stream, ProviderFunCallRequest providerFunCallRequest,
            Map<String, Object> message, Map<String, Object> funCall, Object args, String name, String id) throws Exception {
        Method m = MethodUtils.getMatchingMethod(OpenAiStreamFunCall.class, "afterCreateFunRequest",
                ProviderFunCallRequest.class, Map.class, Map.class, Object.class, String.class, String.class);
        m.setAccessible(true);
        m.invoke(stream, providerFunCallRequest, message, funCall, args, name, id);
    }

    private static void invokeAfterUpdateFunRequest(OpenAiStreamFunCall stream, ProviderFunCallRequest providerFunCallRequest,
            Map<String, Object> message, Map<String, Object> funCall, Object args, String name, String id) throws Exception {
        Method m = MethodUtils.getMatchingMethod(OpenAiStreamFunCall.class, "afterUpdateFunRequest",
                ProviderFunCallRequest.class, Map.class, Map.class, Object.class, String.class, String.class);
        m.setAccessible(true);
        m.invoke(stream, providerFunCallRequest, message, funCall, args, name, id);
    }
}
