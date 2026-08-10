package ai.open.right.workflow.flow.llm.provider.seedream;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.provider.ProviderReader;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.signal.SignalExecutor;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.apache.commons.io.FileUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.io.comparator.NameFileComparator;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.io.File;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;

import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
public class SeedreamStreamTest {

    @Test
    public void testCallback() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        SeedreamRequest c = EasyMock.createMock(SeedreamRequest.class);
        c.appendRequest(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        c.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        String content = IOUtils.toString(ResourceUtils.getURL("classpath:Seed_response_atonce.json").openStream(), StandardCharsets.UTF_8);
        SeedreamStream stream = new SeedreamStream(ProviderStreamConfig.<SeedreamRequest>builder()
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
            protected void storeConversation(String content) throws Exception {
            }

            @Override
            protected Boolean atonce(String message) throws Exception {
                Assert.assertEquals(content, message);
                return true;
            }

            @Override
            protected void afterAtOnce() throws Exception {
            }
        };
        stream.callback(content);
    }

    @Test
    public void testCallbackWithDone() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        SeedreamRequest c = EasyMock.createMock(SeedreamRequest.class);
        c.appendRequest(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        c.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        SeedreamStream stream = new SeedreamStream(ProviderStreamConfig.<SeedreamRequest>builder()
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
            protected void storeConversation(String content) throws Exception {
            }

            @Override
            protected Boolean atonce(String message) throws Exception {
                Assert.fail();
                return true;
            }

            @Override
            protected void afterAtOnce() throws Exception {
            }
        };
        stream.callback(ProviderReader.DONE);
    }

    @Test
    public void testAtOnce() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        SeedreamRequest c = EasyMock.createMock(SeedreamRequest.class);
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        String content = IOUtils.toString(ResourceUtils.getURL("classpath:Seed_response_atonce.json").openStream(), StandardCharsets.UTF_8);
        SeedreamStream stream = new SeedreamStream(ProviderStreamConfig.<SeedreamRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(new TokenStatistic() {

            @Override
            public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
                Assert.assertEquals(Integer.valueOf(15048), tokenData.getTotal());
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
        stream.atonce(content);
        Assert.assertEquals("https://ark-content-generation-v2-cn-beijing.tos-cn-beijing.volces.com/doubao-seedream-4-5/021768564201358943be5294e080788be4ca008f262f4f3be6c6a_0.jpeg?X-Tos-Algorithm=TOS4-HMAC-SHA256&X-Tos-Credential=MOCK_AK_PLACEHOLDER%2F20260116%2Fcn-beijing%2Ftos%2Frequest&X-Tos-Date=20260116T115013Z&X-Tos-Expires=86400&X-Tos-Signature=c251328a021fd99ebe287b320d12a9fe637f02b433e390035be70753a58a7cc2&X-Tos-SignedHeaders=host&x-tos-process=image%2Fwatermark%2Cimage_YXNzZXRzL3dhdGVybWFya192MS5wbmc_eC10b3MtcHJvY2Vzcz1pbWFnZS9yZXNpemUscF8xMzg%3D%2Cx_121%2Cy_121\n", stream.getContentBuffer().toString());
        EasyMock.verify(s, c);
    }

    @Test
    public void testAtOnceBase64() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        SeedreamRequest c = EasyMock.createMock(SeedreamRequest.class);
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        String content = IOUtils.toString(ResourceUtils.getURL("classpath:Seed_response_atonce_base64.json").openStream(), StandardCharsets.UTF_8);
        SeedreamStream stream = new SeedreamStream(ProviderStreamConfig.<SeedreamRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(new TokenStatistic() {

            @Override
            public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
                Assert.assertEquals(Integer.valueOf(15048), tokenData.getTotal());
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
                .mediaInlineService(ObjectBuilder.buildMediaInlineService("http:internal"))
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
        stream.atonce(content);
        Assert.assertEquals("http:internal\n", stream.getContentBuffer().toString());
        EasyMock.verify(s, c);
    }

    @Test
    public void testStream() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        SeedreamRequest c = EasyMock.createMock(SeedreamRequest.class);
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        SeedreamStream stream = new SeedreamStream(ProviderStreamConfig.<SeedreamRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(new TokenStatistic() {

            @Override
            public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
                Assert.assertEquals(Integer.valueOf(46656), tokenData.getTotal());
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
        File rootDir = new File(ResourceUtils.getURL("classpath:Seed_response_stream").getFile());
        File[] dirs = rootDir.listFiles();
        Arrays.sort(dirs, NameFileComparator.NAME_COMPARATOR); // 名字升序排列
        for (File dir : dirs) {
            stream.stream(FileUtils.readFileToString(dir, "UTF-8"));
        }
        Assert.assertEquals("https://ark-content-generation-v2-cn-beijing.tos-cn-beijing.volces.com/doubao-seedream-4-5/0217685709837505ed129591da48e074cf9db74a161503f62c98b_0.jpeg?X-Tos-Algorithm=TOS4-HMAC-SHA256&X-Tos-Credential=MOCK_AK_PLACEHOLDER%2F20260116%2Fcn-beijing%2Ftos%2Frequest&X-Tos-Date=20260116T134331Z&X-Tos-Expires=86400&X-Tos-Signature=f04e54d43611fb35725d0b0cba26c0f61fafe42ec3b5afc265f03e99ff92fa98&X-Tos-SignedHeaders=host\n" +
                "https://ark-content-generation-v2-cn-beijing.tos-cn-beijing.volces.com/doubao-seedream-4-5/0217685709837505ed129591da48e074cf9db74a161503f62c98b_1.jpeg?X-Tos-Algorithm=TOS4-HMAC-SHA256&X-Tos-Credential=MOCK_AK_PLACEHOLDER%2F20260116%2Fcn-beijing%2Ftos%2Frequest&X-Tos-Date=20260116T134332Z&X-Tos-Expires=86400&X-Tos-Signature=69716e710bfcaf2865ac8da22dab2bc51845bef0f88cf2a00db890b4d243f713&X-Tos-SignedHeaders=host\n" +
                "https://ark-content-generation-v2-cn-beijing.tos-cn-beijing.volces.com/doubao-seedream-4-5/0217685709837505ed129591da48e074cf9db74a161503f62c98b_2.jpeg?X-Tos-Algorithm=TOS4-HMAC-SHA256&X-Tos-Credential=MOCK_AK_PLACEHOLDER%2F20260116%2Fcn-beijing%2Ftos%2Frequest&X-Tos-Date=20260116T134333Z&X-Tos-Expires=86400&X-Tos-Signature=414169f0d666c0ed3ece544452aaf7b1bc5d0021abf4a59ca302198de64a5227&X-Tos-SignedHeaders=host\n", stream.getContentBuffer().toString());
    }

    @Test
    public void testStreamBase64() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        SeedreamRequest c = EasyMock.createMock(SeedreamRequest.class);
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        SeedreamStream stream = new SeedreamStream(ProviderStreamConfig.<SeedreamRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(new TokenStatistic() {

            @Override
            public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
                Assert.assertEquals(Integer.valueOf(46656), tokenData.getTotal());
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
                .mediaInlineService(ObjectBuilder.buildMediaInlineService("internal"))
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
        File rootDir = new File(ResourceUtils.getURL("classpath:Seed_response_stream_base64").getFile());
        File[] dirs = rootDir.listFiles();
        Arrays.sort(dirs, NameFileComparator.NAME_COMPARATOR); // 名字升序排列
        for (File dir : dirs) {
            stream.stream(FileUtils.readFileToString(dir, "UTF-8"));
        }
        Assert.assertEquals("https://ark-content-generation-v2-cn-beijing.tos-cn-beijing.volces.com/doubao-seedream-4-5/0217685709837505ed129591da48e074cf9db74a161503f62c98b_0.jpeg?X-Tos-Algorithm=TOS4-HMAC-SHA256&X-Tos-Credential=MOCK_AK_PLACEHOLDER%2F20260116%2Fcn-beijing%2Ftos%2Frequest&X-Tos-Date=20260116T134331Z&X-Tos-Expires=86400&X-Tos-Signature=f04e54d43611fb35725d0b0cba26c0f61fafe42ec3b5afc265f03e99ff92fa98&X-Tos-SignedHeaders=host\n" +
                "https://ark-content-generation-v2-cn-beijing.tos-cn-beijing.volces.com/doubao-seedream-4-5/0217685709837505ed129591da48e074cf9db74a161503f62c98b_1.jpeg?X-Tos-Algorithm=TOS4-HMAC-SHA256&X-Tos-Credential=MOCK_AK_PLACEHOLDER%2F20260116%2Fcn-beijing%2Ftos%2Frequest&X-Tos-Date=20260116T134332Z&X-Tos-Expires=86400&X-Tos-Signature=69716e710bfcaf2865ac8da22dab2bc51845bef0f88cf2a00db890b4d243f713&X-Tos-SignedHeaders=host\n" +
                "internal\n", stream.getContentBuffer().toString());
    }

    /**
     * 覆盖 SeedreamStream#tokenStatistic：body 含 usage（total_tokens），当 total!=0 时调用 tokenStatistic.stat 并设置 segment.usage
     */
    @Test
    public void testTokenStatistic_withUsage_callsStatAndSetsSegmentUsage() throws Exception {
        SeedreamRequest request = new SeedreamRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        SeedreamStream stream = new SeedreamStream(ProviderStreamConfig.<SeedreamRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build());
        Map<String, Object> usage = new HashMap<>();
        usage.put("total_tokens", 100);
        Map<String, Object> body = new HashMap<>();
        body.put("usage", usage);
        stream.tokenStatistic(body);
        Assert.assertNotNull(stream.getSegment().getUsage());
        Assert.assertEquals(Integer.valueOf(100), stream.getSegment().getUsage().getTotal());
    }

    /**
     * 覆盖 SeedreamStream#tokenStatistic：total_tokens==0 时不调用 stat、不设置 segment.usage，不抛异常
     */
    @Test
    public void testTokenStatistic_zeroTotal_doesNotStat() throws Exception {
        SeedreamRequest request = new SeedreamRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        SeedreamStream stream = new SeedreamStream(ProviderStreamConfig.<SeedreamRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build());
        Map<String, Object> usage = new HashMap<>();
        usage.put("total_tokens", 0);
        Map<String, Object> body = new HashMap<>();
        body.put("usage", usage);
        stream.tokenStatistic(body);
        Assert.assertNull(stream.getSegment().getUsage());
    }
}
