package ai.open.right.workflow.flow.llm.provider.anthropic;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.SegmentUsage;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderReader;
import ai.open.right.workflow.flow.llm.provider.ProviderStream;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.token.TokenData;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

@Slf4j
public class AnthropicStream extends ProviderStream<AnthropicRequest> {

    private static final Pattern SSE_PATTERN = Pattern.compile("(?s)^(event:.*?)\\n+(data:.*)$");

    protected Map<Integer, Integer> funCallIndex;

    protected TokenData tokenData;

    public AnthropicStream(ProviderStreamConfig<AnthropicRequest> providerRequestConfig) throws Exception {
        super(providerRequestConfig);
    }

    @Override
    public void callback(String message) throws Exception {
        if (!StringUtils.startsWithIgnoreCase(ProviderReader.DONE, message)) {
            super.callback(message);
        } else if (log.isDebugEnabled()) {
            log.debug("AnthropicStream received the [done]={}", message);
        }
    }

    @Override
    protected Boolean stream(String source) throws Exception {
        // 去掉空行后的数据（必须使用isBlank）
        source = source.lines()
                .filter(line -> !line.isBlank())
                .collect(Collectors.joining("\n"));
        if (StringUtils.isEmpty(source)) {
            return true;
        }
        // Anthropic由两行构成
        Matcher matcher = AnthropicStream.SSE_PATTERN.matcher(source);
        Assert.isTrue(matcher.find() && matcher.groupCount() == 2, "The anthropic message must contain two lines: " + source);
        Assert.isTrue(StringUtils.startsWithIgnoreCase(matcher.group(2), "data:"), "Invalid data");
        Map<String, Object> body = JsonUtils.read(matcher.group(2).replaceFirst("data:", ""), Map.class);
        this.cacheStatistic(body);
        String type = MapUtils.getString(body, "type");
        Boolean finish = false;
        if (StringUtils.equalsIgnoreCase(type, "content_block_delta")) {
            Map<String, Object> delta = MapUtils.getMap(body, "delta");
            String delta_type = MapUtils.getString(delta, "type");
            if (StringUtils.equalsIgnoreCase(delta_type, "thinking_delta")) {
                // 用于子类覆盖
                this.addReason(delta, false);
            } else if (StringUtils.equalsIgnoreCase(delta_type, "input_json_delta")) {
                // Fun Call Input
                this.addFunRequest(body, delta);
            } else if (StringUtils.equalsIgnoreCase(delta_type, "text_delta")) {
                // Markdown格式中，content可能为有意义的空字符
                this.addContent(MapUtils.getString(delta, "text", ""), false);
            }
            this.notify(this.seqid++, false);
        } else if (StringUtils.equalsIgnoreCase(type, "content_block_start")) {
            Map<String, Object> block = MapUtils.getMap(body, "content_block");
            String block_type = MapUtils.getString(block, "type");
            if (StringUtils.equalsIgnoreCase(block_type, "tool_use")) {
                // 处理Fun Call
                this.addFunRequest(body, block);
            }
        } else if (StringUtils.equalsIgnoreCase(type, "message_stop")) {
            this.responseCheck();
            this.addReason(null, true);
            this.addContent("", true);
            // 转换FunCall数据
            this.settleFunRequest();
            // 提交FunCall&记忆
            this.notifyProcess();
            // Anthropic流式会有多个usage累计快照，最终看最后一个即可
            this.tokenStatistic(body);
            finish = true;
        }
        return finish;
    }

    @Override
    protected Boolean atonce(String source) throws Exception {
        Map<String, Object> body = JsonUtils.read(source, Map.class);
        Assert.notNull(body, "Body can not be empty");
        Assert.notNull(body.get("id"), "Body id can not be empty");
        List<Map<String, Object>> content = List.class.cast(body.get("content"));
        Assert.notEmpty(content, "Content can not be empty");
        for (Map<String, Object> each : content) {
            String type = MapUtils.getString(each, "type");
            if (StringUtils.equalsIgnoreCase(type, "thinking")) {
                String thinking = MapUtils.getString(each, "thinking");
                Assert.hasText(thinking, "Thinking can not be empty");
            } else if (StringUtils.equalsIgnoreCase(type, "text")) {
                // Markdown格式中，content可能为有意义的空字符
                this.addContent(MapUtils.getString(each, "text", ""), true);
            } else if (StringUtils.equalsIgnoreCase(type, "tool_use")) {
                // 处理Fun Call
                this.addFunRequest(body, each);
            }
            // 用于子类覆盖
            this.addReason(each, false);
        }
        this.addReason(null, true);
        this.responseCheck();
        this.notifyProcess();
        // 用量统计
        this.cacheStatistic(body);
        this.tokenStatistic(body);
        return true;
    }

    protected void addFunRequest(Map<String, Object> message, Map<String, Object> funCall) throws Exception {
        if (MapUtils.isEmpty(funCall)) {
            return;
        }
        // Index从1开始
        int index = MapUtils.getIntValue(message, "index");
        Object args = MapUtils.getObject(funCall, "input");
        String name = MapUtils.getString(funCall, "name");
        String id = MapUtils.getString(funCall, "id");
        if (!MapUtils.isEmpty(this.funCallIndex) && this.funCallIndex.containsKey(index)) {
            // Update
            this.updateFunRequest(this.providerFunRequests.get(this.funCallIndex.get(index)), message, funCall);
        } else {
            // Create
            this.createFunRequest(message, funCall, index, args, name, id);
        }
    }

    // 用于子类覆盖
    protected void afterUpdateFunRequest(ProviderFunCallRequest providerFunCallRequest, Map<String, Object> message, Map<String, Object> funCall) throws Exception {
    }

    protected void updateFunRequest(ProviderFunCallRequest providerFunCallRequest, Map<String, Object> message, Map<String, Object> funCall) throws Exception {
        Map<String, Object> refer = Map.class.cast(providerFunCallRequest.getRefer());
        String partial = MapUtils.getString(funCall, "partial_json");
        String input = MapUtils.getString(refer, "partial");
        input = input != null ? input + partial : partial;
        refer.put("partial", input);
        providerFunCallRequest.setRefer(refer);
        this.afterUpdateFunRequest(providerFunCallRequest, message, funCall);
    }

    // 用于子类覆盖
    protected void afterCreateFunRequest(ProviderFunCallRequest providerFunCallRequest, Map<String, Object> message, Map<String, Object> funCall, Integer index, Object args, String name, String id) throws Exception {
    }

    protected void createFunRequest(Map<String, Object> message, Map<String, Object> funCall, Integer index, Object args, String name, String id) throws Exception {
        Assert.hasText(name, "Function's name can not be empty");
        Assert.hasText(id, "Function's id can not be empty");
        ProviderFunCallRequest providerFunCallRequest = ProviderFunCallRequest.builder()
                .reason(!StringUtils.isEmpty(this.reasoning) ? this.reasoning.toString() : "")
                .model(this.request.getModel())
                .api(this.request.getApi())
                .refer(funCall)
                .name(name)
                .args(args)
                .id(id)
                .build();
        this.afterCreateFunRequest(providerFunCallRequest, message, funCall, index, args, name, id);
        this.addFunRequest(providerFunCallRequest);
        this.indexFunRequest(index);
    }

    protected void indexFunRequest(Integer index) throws Exception {
        this.funCallIndex = this.funCallIndex != null ? this.funCallIndex : new HashMap<Integer, Integer>();
        this.funCallIndex.put(index, this.providerFunRequests.size() - 1);
    }

    protected void settleFunRequest() throws Exception {
        if (!CollectionUtils.isEmpty(this.providerFunRequests)) {
            for (ProviderFunCallRequest providerFunCallRequest : this.providerFunRequests) {
                Map<String, Object> refer = Map.class.cast(providerFunCallRequest.getRefer());
                if (!MapUtils.isEmpty(refer)) {
                    Object partial = refer.remove("partial");
                    if (partial != null) {
                        // 更新Input
                        Object args = JsonUtils.transfer(partial, Map.class);
                        providerFunCallRequest.setArgs(args);
                        refer.put("input", args);
                    }
                }
            }
        }
    }

    @Override
    // 子类覆盖
    protected void addReason(Map<String, Object> message, Boolean finished) throws Exception {
        String reasoning = MapUtils.getString(message, "thinking");
        if (!StringUtils.isEmpty(reasoning)) {
            // 追加Reasoning
            this.reasoning = this.reasoning != null ? this.reasoning : new StringBuffer();
            this.reasoning.append(reasoning);
            if (this.request.getPrintReason()) {
                // 输出Reason给终端会（默认）记录在多轮会话中，产生双倍容量（reasoning_content本身会记录属性）
                // 可通过子类覆盖自定义实现
                this.addContent(this.providerReason.reason(this.request, reasoning, finished, this.reasonIdx++), false);
            }
        }
    }

    @Override
    protected void tokenStatistic(Map<String, Object> body) throws Exception {
        if (this.tokenData != null) {
            this.tokenStatistic.stat(this.request, this.tokenData);
            this.segment.setUsage(new SegmentUsage(this.tokenData));
            if (log.isInfoEnabled()) {
                log.info("The token statistic={}, total={}, cache={}, thinking={}", this.tokenStatistic, this.tokenData.getTotal(), this.tokenData.getCache(), this.tokenData.getThinking());
            }
        }
    }

    protected void cacheStatistic(Map<String, Object> body) throws Exception {
        // 总input = input_tokens + cache_creation_input_tokens + cache_read_input_tokens
        // 总total = input_tokens + cache_creation_input_tokens + cache_read_input_tokens + output_tokens
        // 总cache = cache_creation_input_tokens + cache_read_input_tokens
        // 先usage然后message.usage
        Map<String, Object> usage = MapUtils.getMap(body, "usage");
        usage = usage != null ? usage : MapUtils.getMap(MapUtils.getMap(body, "message"), "usage");
        Integer thinking = MapUtils.getInteger(MapUtils.getMap(usage, "output_tokens_details"), "thinking_tokens", 0);
        Integer input = MapUtils.getInteger(usage, "input_tokens", 0) + MapUtils.getInteger(usage, "cache_creation_input_tokens", 0) + MapUtils.getInteger(usage, "cache_read_input_tokens", 0);
        Integer cache = MapUtils.getInteger(usage, "cache_creation_input_tokens", 0) + MapUtils.getInteger(usage, "cache_read_input_tokens", 0);
        Integer total = input + MapUtils.getInteger(usage, "output_tokens", 0);
        if (thinking != 0 || input != 0 || cache != 0 || total != 0) {
            this.tokenData = TokenData.builder()
                    .thinking(thinking)
                    .input(input)
                    .cache(cache)
                    .total(total)
                    .build();
        }
    }
}
