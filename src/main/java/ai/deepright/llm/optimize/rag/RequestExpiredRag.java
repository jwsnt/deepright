package ai.deepright.llm.optimize.rag;

import ai.deepright.llm.provider.RequestContextBuilder;
import ai.deepright.utils.TemplateChecker;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.flow.llm.store.history.History;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.io.IOUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.util.List;
import java.util.concurrent.TimeUnit;

@Slf4j
@Setter
@Getter
public class RequestExpiredRag extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_expired";

    protected ResourceService resourceService;

    protected String template4expired;

    protected Integer expired;

    @PostConstruct
    public void init() throws Exception {
        this.template4expired = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4expired).openStream()), StandardCharsets.UTF_8);
        Assert.hasText(this.template4expired, "The template expired must not be empty");
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        this.buildExpired(ragConfig, ragData);
        return new RagAtOnce(ragConfig);
    }

    protected String buildContent(RagConfig ragConfig, RagData ragData, Long expired) throws Exception {
        String template = this.template4expired.replace("#expired", String.valueOf(TimeUnit.MINUTES.convert(expired, TimeUnit.MILLISECONDS)));
        if (log.isWarnEnabled() && !TemplateChecker.check(template)) {
            log.warn("The template contains unexpected characters, please check: {}", template);
        }
        return template;
    }

    protected void buildExpired(RagConfig ragConfig, RagData ragData) throws Exception {
        // History内部不一定有序
        List<History> histories = ragData.getQuery().getHistories();
        if (!CollectionUtils.isEmpty(histories)) {
            Integer lastIndex = null;
            // 找出最后大于指定过期时间的History索引
            for (int index = 0; index < histories.size(); index++) {
                History current = histories.get(index);
                // 服务端会话型记录 且超过指定时间
                if (History.REFERENCE_SERVER.equals(current.getReference()) && History.FUN_CHAT.equals(current.getFunction())) {
                    if ((ragData.getQuery().getCreated() - current.getCreated()) > this.expired) {
                        // 所有server chat中已过期的最新一条
                        if (lastIndex == null || current.getCreated() > histories.get(lastIndex).getCreated()) {
                            lastIndex = index;
                        }
                    }
                }
            }
            if (lastIndex != null) {
                History history = histories.get(lastIndex);
                Long expired = System.currentTimeMillis() - history.getCreated();
                histories.add(RequestContextBuilder.buildContext(ragData.getRequest(), this.buildContent(ragConfig, ragData, expired), History.ROLE_ASSISTANT, history.getCreated() + 1));
            }
        }
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Value("${optimize.expired.template:classpath:config/optimize/expired.md}")
        protected String template4expired;

        @Value("${optimize.expired:600000}")
        protected Integer expired;

        @Bean(RequestExpiredRag.RAG_KEY)
        @ConditionalOnMissingBean(name = RequestExpiredRag.RAG_KEY)
        public RequestExpiredRag requestExpiredRag() throws Exception {
            RequestExpiredRag requestExpiredRag = new RequestExpiredRag();
            BeanUtils.copyProperties(this, requestExpiredRag);
            log.info("RequestExpiredRag inited, timeout4Condition={}", requestExpiredRag.getTimeout4Condition());
            return requestExpiredRag;
        }
    }
}
