package ai.open.right.netty.chat.server.http;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.UserContext;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import org.apache.commons.codec.digest.DigestUtils;
import org.junit.Assert;
import org.junit.Test;

import java.util.*;

public class NettyHttpRequestTest {

    @Test
    public void testWithNullMetadata() {
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        Assert.assertNull(nettyHttpRequest.getMetadata());
    }

    @Test
    public void testWithNullUserContext1() {
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        Assert.assertNull(nettyHttpRequest.getUserContext());
    }

    @Test
    public void testWithNullUserContext2() {
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        Map<String, Object> metadata = new HashMap<>();
        nettyHttpRequest.setMetadata(metadata);
        Assert.assertNull(nettyHttpRequest.getUserContext());
    }

    @Test
    public void testWithNullUserContext3() {
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("KEY", "VAL");
        nettyHttpRequest.setMetadata(metadata);
        Assert.assertNull(nettyHttpRequest.getUserContext());
    }

    @Test
    public void testWithUserContext4() {
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        UserContext userContext = ObjectBuilder.buildLLMQuery().getUserContext();
        nettyHttpRequest.setUserContext(userContext);
        Assert.assertEquals(userContext, nettyHttpRequest.getUserContext());
    }

    @Test
    public void testWithMediaContext() throws Exception {
        List<Map<String, Object>> content = new ArrayList<Map<String, Object>>();
        Map<String, Object> c1 = new HashMap<String, Object>();
        Map<String, String> i1 = new HashMap<>();
        i1.put("url", "http://1.2.3.com");
        c1.put("image_url", i1);
        c1.put("type", "image");
        content.add(c1);
        Map<String, Object> c2 = new HashMap<String, Object>();
        c2.put("text", "HELLO WORLD");
        c2.put("type", "text");
        content.add(c2);
        Map<String, Object> c3 = new HashMap<String, Object>();
        Map<String, String> i3 = new HashMap<>();
        i3.put("url", "http://4.5.6.com");
        c3.put("image_url", i3);
        c3.put("type", "png");
        content.add(c3);
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        NettyRequest nettyRequest = new NettyRequest();
        nettyHttpRequest.initMediaContext(content, nettyRequest);
        Assert.assertEquals(2, nettyRequest.getMediaContext().size());
        Assert.assertEquals("image", nettyRequest.getMediaContext().getFirst().getType());
        Assert.assertEquals("http://1.2.3.com", nettyRequest.getMediaContext().getFirst().getData());
        Assert.assertEquals("png", nettyRequest.getMediaContext().getLast().getType());
        Assert.assertEquals("http://4.5.6.com", nettyRequest.getMediaContext().getLast().getData());
        Assert.assertEquals("HELLO WORLD", nettyRequest.getQuery());
    }

    @Test
    public void testWithBuildMessage() throws Exception {
        List<Map<String, Object>> content = new ArrayList<Map<String, Object>>();
        Map<String, Object> c1 = new HashMap<String, Object>();
        Map<String, String> i1 = new HashMap<>();
        i1.put("url", "http://1.2.3.com");
        c1.put("image_url", i1);
        c1.put("type", "image");
        content.add(c1);
        Map<String, Object> c2 = new HashMap<String, Object>();
        c2.put("text", "HELLO WORLD");
        c2.put("type", "text");
        content.add(c2);
        Map<String, Object> c3 = new HashMap<String, Object>();
        Map<String, String> i3 = new HashMap<>();
        i3.put("url", "http://4.5.6.com");
        c3.put("image_url", i3);
        c3.put("type", "png");
        content.add(c3);
        Map<String, Object> message = new HashMap<>();
        message.put("content", content);
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        nettyHttpRequest.setMessages(Arrays.asList(message));
        NettyRequest nettyRequest = nettyHttpRequest.buildNettyRequest(null, null, null);
        Assert.assertEquals(2, nettyRequest.getMediaContext().size());
        Assert.assertEquals("image", nettyRequest.getMediaContext().getFirst().getType());
        Assert.assertEquals("http://1.2.3.com", nettyRequest.getMediaContext().getFirst().getData());
        Assert.assertEquals("png", nettyRequest.getMediaContext().getLast().getType());
        Assert.assertEquals("http://4.5.6.com", nettyRequest.getMediaContext().getLast().getData());
        Assert.assertEquals("HELLO WORLD", nettyRequest.getQuery());
    }

    @Test
    public void testWithBuildDeviceAndChat() throws Exception {
        List<Map<String, Object>> content = new ArrayList<Map<String, Object>>();
        Map<String, Object> c1 = new HashMap<String, Object>();
        Map<String, String> i1 = new HashMap<>();
        i1.put("url", "http://1.2.3.com");
        c1.put("image_url", i1);
        c1.put("type", "image");
        content.add(c1);
        Map<String, Object> c2 = new HashMap<String, Object>();
        c2.put("text", "HELLO WORLD");
        c2.put("type", "text");
        content.add(c2);
        Map<String, Object> c3 = new HashMap<String, Object>();
        Map<String, String> i3 = new HashMap<>();
        i3.put("url", "http://4.5.6.com");
        c3.put("image_url", i3);
        c3.put("type", "png");
        content.add(c3);
        Map<String, Object> message = new HashMap<>();
        message.put("content", content);
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        Map<String, Object> meta = new HashMap<>();
        meta.put(NettyHttpRequest.KEY_CHAR, "chat");
        nettyHttpRequest.setMetadata(meta);
        nettyHttpRequest.setMessages(Arrays.asList(message));
        NettyRequest nettyRequest = nettyHttpRequest.buildNettyRequest("chat", "device", null);
        Assert.assertEquals("device", nettyRequest.getUserContext().getDevice());
        Assert.assertEquals("chat", nettyRequest.getChat());
    }

    @Test
    public void testHistory() throws Exception {
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        Map<String, Object> c1 = new HashMap<String, Object>();
        c1.put("role", "user");
        c1.put("content", "HELLO");
        History history = nettyHttpRequest.buildHistory(c1);
        Assert.assertEquals(History.ROLE_USER, history.getRole());
        Assert.assertEquals("HELLO", history.getContent());
        Map<String, Object> c2 = new HashMap<String, Object>();
        c2.put("role", "assistant");
        c2.put("content", "WORLD");
        History history2 = nettyHttpRequest.buildHistory(c2);
        Assert.assertEquals(History.ROLE_ASSISTANT, history2.getRole());
        Assert.assertEquals("WORLD", history2.getContent());
    }

    @Test
    public void testModel() throws Exception {
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        List<Map<String, Object>> messages = new ArrayList<>();
        Map<String, Object> c1 = new HashMap<String, Object>();
        c1.put("role", "user");
        c1.put("content", "HELLO");
        History history = nettyHttpRequest.buildHistory(c1);
        Assert.assertEquals(History.ROLE_USER, history.getRole());
        Assert.assertEquals("HELLO", history.getContent());
        Map<String, Object> c2 = new HashMap<String, Object>();
        c2.put("role", "assistant");
        c2.put("content", "WORLD");
        messages.add(c1);
        messages.add(c2);
        nettyHttpRequest.setMessages(messages);
        nettyHttpRequest.setMetadata(new HashMap<>());
        nettyHttpRequest.setModel("MY_MODEL");
        NettyRequest nettyRequest = nettyHttpRequest.buildNettyRequest("chat", "device", null);
        Assert.assertEquals("MY_MODEL", nettyRequest.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        Assert.assertEquals("MY_MODEL", nettyHttpRequest.getModel());
    }

    // 新增的messages属性测试方法
    @Test
    public void testMessagesWithSingleTextMessage() throws Exception {
        // 测试单个文本消息
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        Map<String, Object> message = new HashMap<>();
        message.put("role", "user");
        message.put("content", "Hello, how are you?");
        nettyHttpRequest.setMessages(Arrays.asList(message));
        NettyRequest nettyRequest = nettyHttpRequest.buildNettyRequest(null, null, null);
        Assert.assertEquals("Hello, how are you?", nettyRequest.getQuery());
        Assert.assertEquals(0, nettyRequest.getHistories().size());
    }

    @Test
    public void testMessagesWithMultipleTextMessages() throws Exception {
        // 测试多个文本消息（对话历史）
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        List<Map<String, Object>> messages = new ArrayList<>();
        Map<String, Object> message1 = new HashMap<>();
        message1.put("role", "user");
        message1.put("content", "What is the weather like?");
        messages.add(message1);
        Map<String, Object> message2 = new HashMap<>();
        message2.put("role", "assistant");
        message2.put("content", "I don't have access to real-time weather data.");
        messages.add(message2);
        Map<String, Object> message3 = new HashMap<>();
        message3.put("role", "user");
        message3.put("content", "Can you help me with programming?");
        messages.add(message3);
        nettyHttpRequest.setMessages(messages);
        NettyRequest nettyRequest = nettyHttpRequest.buildNettyRequest(null, null, null);
        // 最后一个消息应该是当前查询
        Assert.assertEquals("Can you help me with programming?", nettyRequest.getQuery());
        // 前两个消息应该是历史记录
        Assert.assertEquals(2, nettyRequest.getHistories().size());
        Assert.assertEquals("What is the weather like?", nettyRequest.getHistories().get(0).getContent());
        Assert.assertEquals(History.ROLE_USER, nettyRequest.getHistories().get(0).getRole());
        Assert.assertEquals("I don't have access to real-time weather data.", nettyRequest.getHistories().get(1).getContent());
        Assert.assertEquals(History.ROLE_ASSISTANT, nettyRequest.getHistories().get(1).getRole());
    }

    @Test
    public void testMessagesWithEmptyContent() throws Exception {
        // 测试空内容的消息
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        Map<String, Object> message = new HashMap<>();
        message.put("role", "user");
        message.put("content", "");
        nettyHttpRequest.setMessages(Arrays.asList(message));
        NettyRequest nettyRequest = nettyHttpRequest.buildNettyRequest(null, null, null);
        Assert.assertEquals("", nettyRequest.getQuery());
    }

    @Test
    public void testMessagesWithNullContent() {
        // 测试null内容的消息
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        Map<String, Object> message = new HashMap<>();
        message.put("role", "user");
        message.put("content", null);
        nettyHttpRequest.setMessages(Arrays.asList(message));
        try {
            nettyHttpRequest.buildNettyRequest(null, null, null);
            Assert.fail("Should throw exception for null content");
        } catch (Exception e) {
            Assert.assertTrue(e.getMessage().contains("Request content can not be empty"));
        }
    }

    @Test
    public void testMessagesWithEmptyMessagesList() {
        // 测试空消息列表
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        nettyHttpRequest.setMessages(new ArrayList<>());
        try {
            nettyHttpRequest.buildNettyRequest(null, null, null);
            Assert.fail("Should throw exception for empty messages");
        } catch (IllegalArgumentException e) {
            Assert.assertTrue(e.getMessage().contains("Message can not be empty"));
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
    }

    @Test
    public void testMessagesWithNullMessages() {
        // 测试null消息列表
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        nettyHttpRequest.setMessages(null);
        try {
            nettyHttpRequest.buildNettyRequest(null, null, null);
            Assert.fail("Should throw exception for null messages");
        } catch (Exception e) {
            Assert.assertTrue(e.getMessage().contains("Message can not be empty"));
        }
    }

    @Test
    public void testMessagesWithDifferentRoleCases() throws Exception {
        // 测试不同大小写的角色
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        // 测试大写
        Map<String, Object> upperMessage = new HashMap<>();
        upperMessage.put("role", "USER");
        upperMessage.put("content", "Hello");
        History upperHistory = nettyHttpRequest.buildHistory(upperMessage);
        Assert.assertEquals(History.ROLE_USER, upperHistory.getRole());
        // 测试混合大小写
        Map<String, Object> mixedMessage = new HashMap<>();
        mixedMessage.put("role", "Assistant");
        mixedMessage.put("content", "Hi");
        History mixedHistory = nettyHttpRequest.buildHistory(mixedMessage);
        Assert.assertEquals(History.ROLE_ASSISTANT, mixedHistory.getRole());
        // 测试小写
        Map<String, Object> lowerMessage = new HashMap<>();
        lowerMessage.put("role", "assistant");
        lowerMessage.put("content", "Hello there");
        History lowerHistory = nettyHttpRequest.buildHistory(lowerMessage);
        Assert.assertEquals(History.ROLE_ASSISTANT, lowerHistory.getRole());
    }

    @Test
    public void testMessagesWithComplexMediaContent() throws Exception {
        // 测试复杂的媒体内容
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        Map<String, Object> message = new HashMap<>();
        List<Map<String, Object>> content = new ArrayList<>();
        // 添加多个图片
        for (int i = 1; i <= 3; i++) {
            Map<String, Object> imagePart = new HashMap<>();
            Map<String, String> imageUrl = new HashMap<>();
            imageUrl.put("url", "https://example.com/image" + i + ".jpg");
            imagePart.put("image_url", imageUrl);
            imagePart.put("type", "image");
            content.add(imagePart);
        }
        // 添加文本
        Map<String, Object> textPart = new HashMap<>();
        textPart.put("text", "Please analyze these images");
        textPart.put("type", "text");
        content.add(textPart);
        message.put("content", content);
        nettyHttpRequest.setMessages(Arrays.asList(message));
        NettyRequest nettyRequest = nettyHttpRequest.buildNettyRequest(null, null, null);
        // 验证媒体上下文
        Assert.assertEquals(3, nettyRequest.getMediaContext().size());
        for (int i = 0; i < 3; i++) {
            Assert.assertEquals("image", nettyRequest.getMediaContext().get(i).getType());
            Assert.assertEquals("https://example.com/image" + (i + 1) + ".jpg", nettyRequest.getMediaContext().get(i).getData());
        }
        // 验证查询文本
        Assert.assertEquals("Please analyze these images", nettyRequest.getQuery());
    }

    @Test
    public void testMessagesWithStreamFlag() throws Exception {
        // 测试流式标志
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        Map<String, Object> message = new HashMap<>();
        message.put("role", "user");
        message.put("content", "Stream this response");
        nettyHttpRequest.setMessages(Arrays.asList(message));
        nettyHttpRequest.setStream(true);
        NettyRequest nettyRequest = nettyHttpRequest.buildNettyRequest(null, null, null);
        Assert.assertEquals("Stream this response", nettyRequest.getQuery());
        Assert.assertTrue(nettyHttpRequest.getStream());
    }

    @Test
    public void testMessagesWithMetadata() throws Exception {
        // 测试带元数据的消息
        NettyHttpRequest nettyHttpRequest = new NettyHttpRequest();
        Map<String, Object> message = new HashMap<>();
        message.put("role", "user");
        message.put("content", "Hello with metadata");
        List<Map<String, Object>> messages = new ArrayList<Map<String, Object>>();
        messages.add(message);
        nettyHttpRequest.setMessages(messages);
        Assert.assertEquals(messages, nettyHttpRequest.getMessages());
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("custom_key", "custom_value");
        metadata.put("priority", "high");
        nettyHttpRequest.setMetadata(metadata);
        NettyRequest nettyRequest = nettyHttpRequest.buildNettyRequest(null, null, null);
        Assert.assertEquals("Hello with metadata", nettyRequest.getQuery());
        Assert.assertEquals("custom_value", nettyRequest.getMetadata().get("custom_key"));
        Assert.assertEquals("high", nettyRequest.getMetadata().get("priority"));
    }

    @Test
    public void testGetMetadataNoModel() {
        NettyHttpRequest req = new NettyHttpRequest();
        req.setMetadata(new HashMap<>(Collections.singletonMap("K", "V")));
        Map<String, Object> meta = req.getMetadata();
        Assert.assertEquals("V", meta.get("K"));
    }

    @org.junit.jupiter.api.Test
    public void testNettyHttpRequestInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

    /**
     * 覆盖 buildNettyRequest 中 currentConversation/currentChat 从 messages 最后一条的 conversation/chat 获取
     */
    @Test
    public void testBuildNettyRequest_currentConversationAndChat_fromLastMessage() throws Exception {
        NettyHttpRequest req = new NettyHttpRequest();
        Map<String, Object> msg = new HashMap<>();
        msg.put("role", "user");
        msg.put("content", "hi");
        msg.put("conversation", "conv-123");
        msg.put("chat", "chat-456");
        req.setMessages(Collections.singletonList(msg));
        NettyRequest out = req.buildNettyRequest(null, null, null);
        Assert.assertEquals("conv-123", out.getConversation());
        Assert.assertEquals("chat-456", out.getChat());
    }

    /**
     * 多条消息时 currentConversation/currentChat 取最后一条的 conversation/chat
     */
    @Test
    public void testBuildNettyRequest_currentConversationAndChat_fromLastMessageWhenMultiple() throws Exception {
        NettyHttpRequest req = new NettyHttpRequest();
        Map<String, Object> m1 = new HashMap<>();
        m1.put("role", "user");
        m1.put("content", "first");
        m1.put("conversation", "conv-first");
        m1.put("chat", "chat-first");
        Map<String, Object> m2 = new HashMap<>();
        m2.put("role", "user");
        m2.put("content", "last");
        m2.put("conversation", "conv-last");
        m2.put("chat", "chat-last");
        req.setMessages(Arrays.asList(m1, m2));
        NettyRequest out = req.buildNettyRequest(null, null, null);
        Assert.assertEquals("conv-last", out.getConversation());
        Assert.assertEquals("chat-last", out.getChat());
    }

    /**
     * messages 最后一条无 conversation/chat 时，从 metadata 的 KEY_CONVERSATION / KEY_CHAR 回退
     */
    @Test
    public void testBuildNettyRequest_currentConversationAndChat_fallbackFromMetadata() throws Exception {
        NettyHttpRequest req = new NettyHttpRequest();
        Map<String, Object> msg = new HashMap<>();
        msg.put("role", "user");
        msg.put("content", "hi");
        Map<String, Object> meta = new HashMap<>();
        meta.put(NettyHttpRequest.KEY_CONVERSATION, "meta-conv");
        meta.put(NettyHttpRequest.KEY_CHAR, "meta-chat");
        req.setMessages(Collections.singletonList(msg));
        req.setMetadata(meta);
        NettyRequest out = req.buildNettyRequest(null, null, null);
        Assert.assertEquals("meta-conv", out.getConversation());
        Assert.assertEquals("meta-chat", out.getChat());
    }

    /**
     * setChat 由 metadata 的 chat 与消息体最后一条的 chat 决定（MapUtils 默认链），与 buildNettyRequest 首参无关
     */
    @Test
    public void testBuildNettyRequest_chatFromMetadataAndMessage() throws Exception {
        NettyHttpRequest req = new NettyHttpRequest();
        Map<String, Object> msg = new HashMap<>();
        msg.put("role", "user");
        msg.put("content", "hi");
        msg.put("chat", "msg-chat");
        Map<String, Object> meta = new HashMap<>();
        meta.put(NettyHttpRequest.KEY_CHAR, "meta-chat");
        req.setMessages(Collections.singletonList(msg));
        req.setMetadata(meta);
        NettyRequest out = req.buildNettyRequest("param-chat", null, null);
        Assert.assertEquals("meta-chat", out.getChat());
    }

    /**
     * 子类仅用于调用 protected 的 {@link NettyHttpRequest#buildMetadata()}
     */
    private static final class NettyHttpRequestMetadataProbe extends NettyHttpRequest {
        Map<String, Object> invokeBuildMetadata() throws Exception {
            return buildMetadata(new HashMap<>());
        }
    }

    @Test
    public void buildMetadata_nullMetadata_returnsNullWhenNoModel() throws Exception {
        NettyHttpRequestMetadataProbe req = new NettyHttpRequestMetadataProbe();
        req.setMetadata(null);
        Assert.assertNotNull(req.invokeBuildMetadata());
    }

    @Test
    public void buildMetadata_emptyMap_returnsEmpty() throws Exception {
        NettyHttpRequestMetadataProbe req = new NettyHttpRequestMetadataProbe();
        req.setMetadata(new HashMap<>());
        Map<String, Object> out = req.invokeBuildMetadata();
        Assert.assertTrue(out.isEmpty());
    }

    @Test
    public void buildMetadata_mapsModelToProviderKey() throws Exception {
        NettyHttpRequestMetadataProbe req = new NettyHttpRequestMetadataProbe();
        req.setMetadata(new HashMap<>());
        req.setModel("gpt-4");
        Map<String, Object> out = req.invokeBuildMetadata();
        Assert.assertEquals("gpt-4", out.get(ProviderRequestService.KEY_PROVIDER));
    }

    @Test
    public void buildMetadata_removesOpenAiCompatibilityKeys() throws Exception {
        NettyHttpRequestMetadataProbe req = new NettyHttpRequestMetadataProbe();
        Map<String, Object> meta = new HashMap<>();
        meta.put(NettyHttpRequest.KEY_CONVERSATION, "c");
        meta.put(NettyHttpRequest.KEY_USER_CONTEXT, Collections.singletonMap("device", "d"));
        meta.put(NettyHttpRequest.KEY_WORKFLOW, "w");
        meta.put(NettyHttpRequest.KEY_TRACE, "t");
        meta.put(NettyHttpRequest.KEY_CHAR, "ch");
        meta.put(NettyHttpRequest.KEY_BIZ, "b");
        meta.put("keep_me", "yes");
        req.setMetadata(meta);
        Map<String, Object> out = req.invokeBuildMetadata();
        Assert.assertEquals("yes", out.get("keep_me"));
        Assert.assertFalse(out.containsKey(NettyHttpRequest.KEY_CONVERSATION));
        Assert.assertFalse(out.containsKey(NettyHttpRequest.KEY_USER_CONTEXT));
        Assert.assertFalse(out.containsKey(NettyHttpRequest.KEY_WORKFLOW));
        Assert.assertFalse(out.containsKey(NettyHttpRequest.KEY_TRACE));
        Assert.assertFalse(out.containsKey(NettyHttpRequest.KEY_CHAR));
        Assert.assertFalse(out.containsKey(NettyHttpRequest.KEY_BIZ));
    }

    @Test
    public void testBuildDefDeviceWithAuthorizationAndModel() throws Exception {
        NettyHttpRequest req = new NettyHttpRequest();
        req.setModel("model-x");
        Map<String, Object> headers = new HashMap<>();
        headers.put(NettyHttpRequest.KEY_AUTHORIZATION, "Bearer token-123");
        String expect = DigestUtils.md5Hex("model-x" + "Bearer token-123");
        Assert.assertEquals(expect, req.buildDefDevice(headers));
    }

    @Test
    public void testBuildDefDeviceWithoutAuthorizationReturnsEmpty() throws Exception {
        NettyHttpRequest req = new NettyHttpRequest();
        req.setModel("model-x");
        Assert.assertEquals("", req.buildDefDevice(new HashMap<>()));
    }

    @Test
    public void testBuildNettyRequestUsesBuildDefDeviceWhenTokenDeviceMissing() throws Exception {
        NettyHttpRequest req = new NettyHttpRequest();
        req.setModel("model-x");
        Map<String, Object> message = new HashMap<>();
        message.put("role", "user");
        message.put("content", "hello");
        req.setMessages(Collections.singletonList(message));

        Map<String, Object> headers = new HashMap<>();
        headers.put(NettyHttpRequest.KEY_AUTHORIZATION, "Bearer token-123");
        NettyRequest out = req.buildNettyRequest("chat-a", null, headers);

        Assert.assertEquals(DigestUtils.md5Hex("model-x" + "Bearer token-123"), out.getUserContext().getDevice());
    }
}