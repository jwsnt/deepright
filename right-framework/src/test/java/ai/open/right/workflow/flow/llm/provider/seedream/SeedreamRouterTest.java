package ai.open.right.workflow.flow.llm.provider.seedream;

import ai.open.right.ObjectBuilder;
import ai.open.right.config.HttpClientConfig;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.media.MediaContext;
import org.apache.http.client.methods.HttpPost;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.*;

/**
 * SeedRouter 单测，覆盖 reConfig, url, body 以及内部类。
 */
public class SeedreamRouterTest {

    private SeedreamRouter seedRouter;

    @BeforeEach
    public void setUp() {
        seedRouter = new SeedreamRouter();
        seedRouter.setUrl("http://test-url");
    }

    @Test
    public void testReConfig() throws Exception {
        SeedreamRequest request = EasyMock.createMock(SeedreamRequest.class);
        LLMConfig config = EasyMock.createMock(LLMConfig.class);
        Message message = EasyMock.createMock(Message.class);
        HttpPost httpPost = new HttpPost("http://test-url");
        request.setFunCallTimeout(EasyMock.anyInt());
        request.setTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getToken()).andReturn("test-token").anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        // 修复 1: mock getMessage()
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        // 修复 1: 为 message mock 添加 isFromFunCall() 的 mock
        EasyMock.expect(message.isFromFunCall()).andReturn(false).anyTimes();
        // 修复: 添加 getUpstream() 和 getTimeout() 的 mock
        EasyMock.expect(message.getUpstream()).andReturn(null).anyTimes();
        EasyMock.expect(config.getTimeout(EasyMock.anyInt())).andReturn(60000).anyTimes();

        EasyMock.replay(request, config, message);
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        seedRouter.setHttpClientConfig(httpClientConfig);
        seedRouter.setTimeout(2000);
        seedRouter.setTimeoutRate(2.0D);
        seedRouter.reConfig(request, config, httpPost);

        // 验证 Header 是否正确添加
        Assertions.assertEquals("test-token", httpPost.getFirstHeader("Authorization").getValue());
        EasyMock.verify(request, config, message);
    }

    @Test
    public void testUrl() throws Exception {
        SeedreamRequest request = EasyMock.createMock(SeedreamRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getUrl()).andReturn("").anyTimes();
        LLMConfig config = EasyMock.createMock(LLMConfig.class);
        EasyMock.replay(request, config);
        String result = seedRouter.url(request, config, "test");
        Assertions.assertEquals("http://test-url", result);
        EasyMock.verify(request, config);
    }

    @Test
    public void testBody() throws Exception {
        SeedreamRequest request = EasyMock.createMock(SeedreamRequest.class);
        Message message = EasyMock.createMock(Message.class);
        List<MediaContext> mediaContext = new ArrayList<>();
        MediaContext mediaContext1 = new MediaContext();
        mediaContext1.setData("http://test-url");
        mediaContext1.setType("image/jpeg");
        mediaContext.add(mediaContext1);
        EasyMock.expect(request.getMediaContext()).andReturn(mediaContext).anyTimes();
        EasyMock.expect(request.getSequentialOptions()).andReturn(null).anyTimes();
        EasyMock.expect(request.getSequential()).andReturn("disable").anyTimes();
        EasyMock.expect(request.getOptimizeOptions()).andReturn(null).anyTimes();
        EasyMock.expect(request.getGuidance()).andReturn(null).anyTimes();
        EasyMock.expect(request.getSize()).andReturn(null).anyTimes();
        EasyMock.expect(request.getSeed()).andReturn(1024).anyTimes();
        EasyMock.expect(request.getMimeType()).andReturn(null).anyTimes();
        EasyMock.expect(request.getImages()).andReturn(1).anyTimes();
        EasyMock.expect(request.getFormat()).andReturn("URL").anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getModel()).andReturn("claude-3").anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(false).anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("hello").anyTimes();
        EasyMock.expect(request.getPrompt()).andReturn("system prompt").anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.replay(request, message);

        Object body = seedRouter.body(request);
        Assertions.assertTrue(body instanceof SeedreamRouter.SeedreamMessage);
        SeedreamRouter.SeedreamMessage seedMessage = (SeedreamRouter.SeedreamMessage) body;

        // 验证字段赋值
        Assertions.assertEquals("claude-3", seedMessage.getModel());
        Assertions.assertEquals("hello", seedMessage.getPrompt());

        EasyMock.verify(request, message);
    }

    @Test
    public void testSeedMessageWithMime() throws Exception {
        SeedreamRequest request = EasyMock.createMock(SeedreamRequest.class);
        Message message = EasyMock.createMock(Message.class);
        MediaContext mediaContext = EasyMock.createMock(MediaContext.class);
        MediaContext mediaContext2 = EasyMock.createMock(MediaContext.class);
        SeedreamMedia seedMedia = EasyMock.createMock(SeedreamMedia.class);
        EasyMock.expect(mediaContext.getType("inline:image/png")).andReturn("inline:image/png").anyTimes();
        EasyMock.expect(mediaContext2.getType("inline:image/png")).andReturn("inline:image/png").anyTimes();
        EasyMock.expect(mediaContext2.getType("image/png")).andReturn("http://url").anyTimes();
        EasyMock.expect(request.getFormat()).andReturn("URL").anyTimes();
        EasyMock.expect(request.getStream()).andReturn(null).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(false).anyTimes();

        EasyMock.expect(request.getSequentialOptions()).andReturn(null).anyTimes();
        EasyMock.expect(request.getSequential()).andReturn("disable").anyTimes();
        EasyMock.expect(request.getOptimizeOptions()).andReturn(null).anyTimes();
        EasyMock.expect(request.getGuidance()).andReturn(null).anyTimes();
        EasyMock.expect(request.getSize()).andReturn(null).anyTimes();
        EasyMock.expect(request.getSeed()).andReturn(1024).anyTimes();
        EasyMock.expect(request.getImages()).andReturn(5).anyTimes();
        // 模拟多媒体上下文 (图片)
        EasyMock.expect(request.hasMimeContext()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMediaContext()).andReturn(Arrays.asList(mediaContext, mediaContext2)).anyTimes();
        EasyMock.expect(request.getSeedMedia()).andReturn(seedMedia).anyTimes();
        // 修复 2: 将媒体类型改为以 inline: 开头
        EasyMock.expect(request.getMimeType()).andReturn("inline:image/png").anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("what is this?").anyTimes();

        // 修复 3: 确保 mediaContext.getType 返回 "inline:image/png"
        EasyMock.expect(mediaContext.getType()).andReturn("inline:image/png").anyTimes();
        // 修复 2: 将 anthropicMedia.getPrefix 和 getKeyUrl 的期望参数改为 "inline:image/png"
        EasyMock.expect(seedMedia.getKeyUrl("inline:image/png")).andReturn("image").anyTimes();
        EasyMock.expect(seedMedia.getPrefix("inline:image/png")).andReturn("data:image/png;base64,").anyTimes();
        EasyMock.expect(seedMedia.getPrefix("image/png")).andReturn("http://url").anyTimes();
        EasyMock.expect(mediaContext.getData()).andReturn("base64data").anyTimes();
        EasyMock.expect(mediaContext2.getData()).andReturn("http://url").anyTimes();
        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();

        EasyMock.replay(request, message, mediaContext, mediaContext2, seedMedia);

        SeedreamRouter.SeedreamMessage seedMessage = new SeedreamRouter.SeedreamMessage(request);
        List<String> content = List.class.cast(seedMessage.getImage());
        Assertions.assertEquals(2, content.size()); // Text + Image
        Assertions.assertEquals("http://urlbase64data", content.get(0));
        Assertions.assertEquals("http://urlhttp://url", content.get(1));
        EasyMock.verify(request, message, mediaContext, mediaContext2, seedMedia);
    }

    @Test
    public void testSeedMessageWithMimeNotInline() throws Exception {
        SeedreamRequest request = EasyMock.createMock(SeedreamRequest.class);
        Message message = EasyMock.createMock(Message.class);
        MediaContext mediaContext = EasyMock.createMock(MediaContext.class);
        EasyMock.expect(mediaContext.getType("inline:image/png")).andReturn("inline:image/png").anyTimes();
        EasyMock.expect(mediaContext.getType("text/plain")).andReturn("text/plain").anyTimes();
        SeedreamMedia seedMedia = EasyMock.createMock(SeedreamMedia.class);
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMediaContext()).andReturn(Arrays.asList(mediaContext)).anyTimes();
        EasyMock.expect(request.getMimeType()).andReturn("text/plain").anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("query").anyTimes();
        EasyMock.expect(request.getImages()).andReturn(5).anyTimes();
        // 模拟非内联媒体 (如 URL)
        EasyMock.expect(mediaContext.getType()).andReturn("text/plain").anyTimes();
        EasyMock.expect(seedMedia.getKeyUrl("text/plain")).andReturn("url").anyTimes();
        EasyMock.expect(mediaContext.getData()).andReturn("http://url").anyTimes();

        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();

        Map<String, Object> sequentialOptions = new HashMap<>();
        EasyMock.expect(request.getSequentialOptions()).andReturn(sequentialOptions).anyTimes();
        Map<String, Object> optimizeOptions = new HashMap<>();
        EasyMock.expect(request.getOptimizeOptions()).andReturn(optimizeOptions).anyTimes();
        EasyMock.expect(request.getSequential()).andReturn("DISABLE").anyTimes();
        EasyMock.expect(request.getGuidance()).andReturn(3.0).anyTimes();
        EasyMock.expect(request.getFormat()).andReturn("URL").anyTimes();
        EasyMock.expect(request.getSeed()).andReturn(1024).anyTimes();
        EasyMock.expect(request.getSize()).andReturn("1*1").anyTimes();
        EasyMock.replay(request, message, mediaContext, seedMedia);
        SeedreamRouter.SeedreamMessage seedMessage = new SeedreamRouter.SeedreamMessage(request);
        Assertions.assertEquals(sequentialOptions, seedMessage.getSequentialOptions());
        Assertions.assertEquals(optimizeOptions, seedMessage.getOptimizeOptions());
        Assertions.assertEquals("DISABLE", seedMessage.getSequential());
        Assertions.assertEquals(3.0, seedMessage.getGuidance());
        Assertions.assertEquals("URL", seedMessage.getFormat());
        Assertions.assertEquals(1024, seedMessage.getSeed());
        Assertions.assertEquals("1*1", seedMessage.getSize());
        String content = String.class.cast(seedMessage.getImage());
        Assertions.assertEquals("http://url", content);
        Assertions.assertEquals(false, seedMessage.getStream());
        EasyMock.verify(request, message, mediaContext, seedMedia);
    }

    @Test
    public void testReader() throws Exception {
        SeedreamRequest request = EasyMock.createMock(SeedreamRequest.class);
        LLMConfig config = EasyMock.createMock(LLMConfig.class);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        Message message = EasyMock.createMock(Message.class); // 修复 3: 添加 message mock
        seedRouter.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        // 修复 3: 为 anthropicRouter 设置 queue 和 buffer 属性
        seedRouter.setQueue(10);
        seedRouter.setTimeout(10);
        seedRouter.setDiscard(10);
        seedRouter.setBuffer(1024);
        seedRouter.setQueueTimeout(1024);
        seedRouter.setCapacity(1024);
        // 修复 3: mock request.getMessage() 和 message.isFromFunCall()
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.isFromFunCall()).andReturn(false).anyTimes();
        // 修复: 添加 getUpstream() 和 getTimeout() 的 mock
        EasyMock.expect(message.getUpstream()).andReturn(null).anyTimes();
        EasyMock.expect(config.getTimeout(EasyMock.anyInt())).andReturn(60000).anyTimes();

        // 测试有图片缓冲区的情况
        EasyMock.expect(config.hasNetworkBuffer()).andReturn(true).anyTimes();
        EasyMock.expect(config.getNetworkBuffer()).andReturn(1024).anyTimes();
        // 修复 4: 确保所有 mock 调用在 replay 之前都已定义
        EasyMock.replay(request, config, callback, message);

        SeedreamReader reader = seedRouter.reader(request, config, callback);
        Assertions.assertNotNull(reader);

        // 测试无图片缓冲区的情况
        EasyMock.reset(config);
        EasyMock.expect(config.hasNetworkBuffer()).andReturn(false).anyTimes();
        // 修复: 重置后重新 mock getTimeout
        EasyMock.expect(config.getTimeout(EasyMock.anyInt())).andReturn(60000).anyTimes();
        EasyMock.replay(config);
        reader = seedRouter.reader(request, config, callback);
        Assertions.assertNotNull(reader);

        // 修复 4: 确保 verify 被调用
        EasyMock.verify(request, config, callback, message);
    }

    @Test
    public void testInitConfig() throws Exception {
        SeedreamRouter.InitConfig initConfig = new SeedreamRouter.InitConfig();
        initConfig.setUrl("http://init-url");
        SeedreamRouter router = initConfig.seedreamRouter();
        Assertions.assertEquals("http://init-url", router.getUrl());
    }
}

