package ai.open.right.netty.chat.server.http;

import ai.open.right.context.UserContext;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.codec.digest.DigestUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Setter
@Getter
@Slf4j
// Open AI协议
public class NettyHttpRequest {

    public static final String KEY_AUTHORIZATION = "Authorization";

    public static final String KEY_CONVERSATION = "conversation";

    public static final String KEY_USER_CONTEXT = "userContext";

    public static final String KEY_WORKFLOW = "workflow";

    public static final String KEY_TRACE = "trace";

    public static final String KEY_CHAR = "chat";

    public static final String KEY_BIZ = "biz";

    protected List<Map<String, Object>> messages;

    protected Map<String, Object> metadata;

    protected UserContext userContext;

    protected Boolean stream = false;

    // 终端选择模型的类型（可选）
    protected String model;

    public NettyRequest buildNettyRequest(String chat, String device, Map<String, Object> headers) throws Exception {
        NettyRequest nettyRequest = new NettyRequest();
        // 至少包含一条Message
        Assert.notEmpty(this.messages, "Message can not be empty, please check request body");
        nettyRequest.setHistories(new ArrayList<History>());
        String currentConversation = null;
        String currentChat = null;
        for (Map<String, Object> each : this.messages) {
            Object content = each.get("content");
            // Content内容不能为空
            Assert.notNull(content, "Request content can not be empty, please check message field");
            if (List.class.isAssignableFrom(content.getClass())) {
                // 初始化Media上下文
                this.initMediaContext((List<Map<String, Object>>) content, nettyRequest);
            } else {
                // 处理普通的多文本Query
                nettyRequest.getHistories().add(this.buildHistory(each));
            }
            // 获取最后一条消息的会话
            currentConversation = MapUtils.getString(each, "conversation");
            currentChat = MapUtils.getString(each, "chat");
        }
        // Text Query，拆分当前Query和History
        if (!CollectionUtils.isEmpty(nettyRequest.getHistories())) {
            // 获取最后的History作为Query，并从Histories移除
            nettyRequest.setQuery(nettyRequest.getHistories().getLast().getContent());
            nettyRequest.getHistories().removeLast();
        }
        nettyRequest.setChat(MapUtils.getString(this.metadata, NettyHttpRequest.KEY_CHAR, StringUtils.defaultIfEmpty(currentChat, chat)));
        nettyRequest.setUserContext(this.buildUserContext(!StringUtils.isEmpty(device) ? device : this.buildDefDevice(headers)));
        nettyRequest.setConversation(MapUtils.getString(this.metadata, NettyHttpRequest.KEY_CONVERSATION, currentConversation));
        nettyRequest.setWorkflow(MapUtils.getString(this.metadata, NettyHttpRequest.KEY_WORKFLOW));
        nettyRequest.setTrace(MapUtils.getString(this.metadata, NettyHttpRequest.KEY_TRACE));
        nettyRequest.setBiz(MapUtils.getString(this.metadata, NettyHttpRequest.KEY_BIZ));
        nettyRequest.setMetadata(this.buildMetadata(headers));
        return nettyRequest;
    }

    // 初始化Media上下文
    protected void initMediaContext(List<Map<String, Object>> content, NettyRequest request) throws Exception {
        for (Map<String, Object> part : content) {
            String type = String.class.cast(part.get("type"));
            if (!MediaContext.TEXT.equalsIgnoreCase(type)) {
                Map<String, Object> data = Map.class.cast(part.get("image_url"));
                data = data != null ? data : Map.class.cast(part.get("file_url"));
                data = data != null ? data : Map.class.cast(part.get("video_url"));
                this.initMediaData(request, data, type);
            } else {
                // Text Query
                String query = String.class.cast(part.get(MediaContext.TEXT));
                Assert.hasText(query, "Field `query` can not be empty: " + part);
                if (log.isDebugEnabled()) {
                    log.debug("Add media query context={}", query);
                }
                request.addQuery(query);
            }
        }
    }

    protected void initMediaData(NettyRequest request, Map<String, Object> part, String type) throws Exception {
        MediaContext media = new MediaContext();
        Assert.notEmpty(part, "Field `xxx_url` can not be empty: " + type);
        String url = MapUtils.getString(part, "url");
        Assert.hasText(url, "Field `url` can not be empty: " + part);
        media.setType(type);
        media.setData(url);
        if (log.isDebugEnabled()) {
            log.debug("Add media url context={}", media);
        }
        request.initMediaContext();
        request.getMediaContext().add(media);
    }

    protected Map<String, Object> buildMetadata(Map<String, Object> headers) throws Exception {
        // 提取 Token
        this.metadata = this.metadata != null ? this.metadata : new HashMap<String, Object>();
        if (!MapUtils.isEmpty(headers)) {
            this.metadata.putAll(headers);
        }
        // Open AI标准参数 model
        if (!StringUtils.isEmpty(this.model)) {
            // 模型映射到服务商
            this.metadata.put(ProviderRequestService.KEY_PROVIDER, this.model);
        }
        // 移除兼容OpenAI的额外数据（已经使用）
        if (!CollectionUtils.isEmpty(this.metadata)) {
            this.metadata.remove(NettyHttpRequest.KEY_AUTHORIZATION);
            this.metadata.remove(NettyHttpRequest.KEY_CONVERSATION);
            this.metadata.remove(NettyHttpRequest.KEY_USER_CONTEXT);
            this.metadata.remove(NettyHttpRequest.KEY_WORKFLOW);
            this.metadata.remove(NettyHttpRequest.KEY_TRACE);
            this.metadata.remove(NettyHttpRequest.KEY_CHAR);
            this.metadata.remove(NettyHttpRequest.KEY_BIZ);
        }
        if (log.isDebugEnabled()) {
            log.debug("Http request metadata={}", this.metadata);
        }
        return this.metadata;
    }

    // 将请求中的Message转为History
    protected History buildHistory(Map<String, Object> message) throws Exception {
        History history = new History();
        String role = String.class.cast(message.get("role"));
        if (StringUtils.containsIgnoreCase("assistant", role)) {
            history.setAssistant();
        } else {
            history.setUser();
        }
        history.setContent(this.buildContent(message));
        if (log.isDebugEnabled()) {
            log.debug("Add history={}", history);
        }
        return history;
    }

    protected String buildContent(Map<String, Object> message) throws Exception {
        Object content = message.get("content");
        return String.class.equals(content.getClass()) ? String.class.cast(content) : JsonUtils.write(content);
    }

    // Device参数为从Token解析的，Metadata.UserContext的Device优先于Token解析
    protected UserContext buildUserContext(String device) throws Exception {
        UserContext userContext = null;
        if (!CollectionUtils.isEmpty(this.metadata)) {
            // 兼容OpenAI SDK不支持独立UserContext，从MetaData获取
            Map<String, String> metaContext = Map.class.cast(this.metadata.get(NettyHttpRequest.KEY_USER_CONTEXT));
            if (!CollectionUtils.isEmpty(metaContext)) {
                userContext = UserContext.builder()
                        .language(metaContext.get("language"))
                        .region(metaContext.get("region"))
                        .system(metaContext.get("system"))
                        .device(metaContext.get("device"))
                        .brand(metaContext.get("brand"))
                        .model(metaContext.get("model"))
                        .token(metaContext.get("token"))
                        .build();
            }
        }
        if (userContext == null) {
            userContext = UserContext.builder()
                    .device(device)
                    .build();
        } else if (StringUtils.isEmpty(userContext.getDevice())) {
            userContext.setDevice(device);
        }
        return userContext;
    }

    protected String buildDefDevice(Map<String, Object> headers) throws Exception {
        // 必需填写APK KEY
        String apiKey = MapUtils.getString(headers, NettyHttpRequest.KEY_AUTHORIZATION);
        return !StringUtils.isEmpty(apiKey) ? DigestUtils.md5Hex(this.model + apiKey) : "";
    }
}