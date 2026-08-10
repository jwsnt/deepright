package ai.open.right.workflow.flow.llm.provider.seedream;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.SegmentUsage;
import ai.open.right.workflow.flow.llm.provider.ProviderReader;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderStream;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.media.MediaInlineData;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.util.List;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

@Slf4j
public class SeedreamStream extends ProviderStream<SeedreamRequest> {

    private static final Pattern SSE_PATTERN = Pattern.compile("(?s)^(event:.*?)\\n+(data:.*)$");

    public SeedreamStream(ProviderStreamConfig<SeedreamRequest> providerRequestConfig) throws Exception {
        super(providerRequestConfig);
    }

    @Override
    public void callback(String message) throws Exception {
        if (!StringUtils.startsWithIgnoreCase(ProviderReader.DONE, message)) {
            super.callback(message);
        } else if (log.isDebugEnabled()) {
            log.debug("SeedStream received the [done]={}", message);
        }
    }

    @Override
    protected Boolean stream(String source) throws Exception {
        Matcher matcher = SeedreamStream.SSE_PATTERN.matcher(source);
        // Seed由两行构成
        Assert.isTrue(matcher.find() && matcher.groupCount() == 2, "The seed message must contain two lines");
        Assert.isTrue(StringUtils.startsWithIgnoreCase(matcher.group(2), "data:"), "Invalid data");
        Map<String, Object> body = JsonUtils.read(matcher.group(2).replaceFirst("data:", ""), Map.class);
        // 首行为结束事件
        Boolean finish = StringUtils.endsWithIgnoreCase(matcher.group(1), "image_generation.completed");
        if (!finish) {
            String path = MapUtils.getString(body, "url");
            if (!StringUtils.isEmpty(path)) {
                this.addContent(this.addPath(this.addUrlData(path)), false);
            } else {
                // b64_json
                this.addContent(this.addPath(this.addInlineData(MapUtils.getString(body, "b64_json"))), false);
            }
            this.notify(this.seqid++, false);
        } else {
            this.addContent("", true);
            this.notifyProcess();
        }
        this.tokenStatistic(body);
        return finish;
    }

    @Override
    protected Boolean atonce(String source) throws Exception {
        Map<String, Object> body = JsonUtils.read(source, Map.class);
        Assert.notNull(body, "Body can not be empty");
        List<Map<String, Object>> data = List.class.cast(MapUtils.getObject(body, "data"));
        Assert.notEmpty(data, "Data can not be empty");
        int index = 0;
        for (Map<String, Object> each : data) {
            String path = MapUtils.getString(each, "url");
            boolean isLast = (index++ == data.size() - 1);
            if (!StringUtils.isEmpty(path)) {
                this.addContent(this.addPath(this.addUrlData(path)), isLast);
            } else {
                // b64_json
                this.addContent(this.addPath(this.addInlineData(MapUtils.getString(each, "b64_json"))), isLast);
            }
        }
        this.notifyProcess();
        // 用量统计
        this.tokenStatistic(body);
        return true;
    }

    protected String addPath(String path) throws Exception {
        return path + System.lineSeparator();
    }

    protected String addUrlData(String data) throws Exception {
        return data;
    }

    protected String addInlineData(String data) throws Exception {
        Assert.notNull(this.mediaInlineService, "The media inline service can not be empty, please config `media.enable`");
        return this.mediaInlineService.write(MediaInlineData.builder()
                .mediaType("image/png")
                .data(data)
                .build(), this.request.getMessage());
    }

    @Override
    protected void tokenStatistic(Map<String, Object> body) throws Exception {
        Map<String, Object> usage = MapUtils.getMap(body, "usage");
        // usage.output_tokens: 生成图片花费的token数量
        // usage.total_tokens: 本次请求消耗的总token数量
        Integer output = MapUtils.getInteger(usage, "output_tokens", 0);
        Integer total = MapUtils.getInteger(usage, "total_tokens", 0);
        if (total != 0) {
            // 任一不为0时记录
            TokenData tokenData = TokenData.builder()
                    .input(total - output)
                    .total(total)
                    .thinking(0)
                    .cache(0)
                    .build();
            this.tokenStatistic.stat(this.request, tokenData);
            this.segment.setUsage(new SegmentUsage(tokenData));
            if (log.isDebugEnabled()) {
                log.debug("The token statistic: total={}", total);
            }
        }
    }
}
