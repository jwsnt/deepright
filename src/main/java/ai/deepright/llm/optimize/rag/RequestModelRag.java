package ai.deepright.llm.optimize.rag;

import ai.deepright.cli.CliPrinter;
import ai.deepright.complex.ComplexityUtils;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.llm.provider.RequestContextUtils;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.notify.Notifier;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

@Slf4j
@Getter
@Setter
public class RequestModelRag extends RagCondition implements RagService {

    public static final String LANG_KEY_REQUEST_CAPACITY = "request.capacity";

    public static final String LANG_KEY_REQUEST_MODEL = "request.model";

    public static final String RAG_KEY = "rag_model";

    protected Double focusRate;

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        String model = ragData.getQuery().delMetadata(RequestModelRag.LANG_KEY_REQUEST_MODEL, String.class);
        Long bytes = ragData.getQuery().delMetadata(RequestModelRag.LANG_KEY_REQUEST_CAPACITY, Long.class);
        if (!StringUtils.isEmpty(model) && bytes != null) {
            if ((bytes / (double) RequestContextUtils.limit(ragData.getQuery(), ragData.getRequest().getModel()) > this.focusRate)) {
                this.notify(ragConfig, ragData, XmlResourceLang.get(RequestModelRag.LANG_KEY_REQUEST_CAPACITY).replace("#size", String.valueOf(bytes / 1024)).replace("#model", model));
                ComplexityUtils.upgrade(ragData.getQuery());
            }
        }
        return new RagAtOnce(ragConfig);
    }

    public void notify(RagConfig ragConfig, RagData ragData, String content) throws Exception {
        if (!FeatureFlag.isSilent(ragData.getQuery())) {
            Segment.SegmentConfig segmentConfig = Segment.SegmentConfig.builder()
                    .metadata(CliPrinter.process(RequestModelRag.RAG_KEY))
                    .content(new StringBuffer(content))
                    .workflow(ragData.getQuery().getWorkflow())
                    .notifier(Notifier.SOURCE)
                    .build();
            this.notifierService.notify(Segment.build(ragData.getQuery(), segmentConfig), ragData.getQuery(), ragData.getQuery());
        }
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Value("${optimize.remind.focus.rate:0.25}")
        protected Double focusRate;

        @Bean(RequestModelRag.RAG_KEY)
        @ConditionalOnMissingBean(name = RequestModelRag.RAG_KEY)
        public RequestModelRag requestCapacity() throws Exception {
            RequestModelRag requestCapacity = new RequestModelRag();
            BeanUtils.copyProperties(this, requestCapacity);
            log.info("RequestModelRag inited, timeout4Condition={}", requestCapacity.getTimeout4Condition());
            return requestCapacity;
        }
    }
}
