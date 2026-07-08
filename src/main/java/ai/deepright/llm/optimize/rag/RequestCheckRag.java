package ai.deepright.llm.optimize.rag;

import static org.springframework.util.ObjectUtils.isEmpty;

import ai.open.right.protocol.ProtocolCode;

import ai.open.right.WorkflowException;

import ai.deepright.feature.FeatureFlag;
import ai.deepright.llm.notifier.MultiSourceNotifier;
import ai.deepright.utils.TemplateChecker;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCall;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.List;

@Slf4j
@Getter
@Setter
public class RequestCheckRag extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_check";

    protected Boolean debug;

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (this.debug) {
            this.checkInitialQuery(ragConfig, ragData);
            this.checkOriginalQuery(ragConfig, ragData);
            this.checkSchemaOnConfig(ragConfig, ragData);
            this.checkSchemaOnRuntime(ragConfig, ragData);
            this.checkPromptPlaceholder(ragConfig, ragData);
            this.checkFunCallPlaceholder(ragConfig, ragData);
        }
        return new RagAtOnce(ragConfig);
    }

    protected void checkOriginalQuery(RagConfig ragConfig, RagData ragData) throws Exception {
        // 不为Task或后台线程时需要包含原始请求内容
        if (StringUtils.containsIgnoreCase(MultiSourceNotifier.MAIN, SplitUtils.join(ragData.getQuery())) && !FeatureFlag.isTask(ragData.getQuery()) && !FeatureFlag.isDaemon(ragData.getQuery())) {
            WorkflowException.check(!(StringUtils.containsIgnoreCase(ragData.getQuery().getQuery(), ragData.getQuery().getOriginal())), "The request query={" + ragData.getQuery().getQuery() + "} must contain original query={" + ragData.getQuery().getOriginal() + "}", ProtocolCode.C400);
        }
    }

    protected void checkInitialQuery(RagConfig ragConfig, RagData ragData) throws Exception {
        // 为Task或后台线程时需要包含初始请求内容
        if (StringUtils.containsIgnoreCase(MultiSourceNotifier.MAIN, SplitUtils.join(ragData.getQuery())) && FeatureFlag.isTask(ragData.getQuery()) || FeatureFlag.isDaemon(ragData.getQuery())) {
            WorkflowException.check(!(StringUtils.containsIgnoreCase(ragData.getQuery().getQuery(), ragData.getQuery().getInitial())), "The request query={" + ragData.getQuery().getQuery() + "} must contain initial query={" + ragData.getQuery().getInitial() + "}", ProtocolCode.C400);
        }
    }

    protected void checkSchemaOnConfig(RagConfig ragConfig, RagData ragData) throws Exception {
        // 如果配置了Schema且为Open AI系列，Prompt必需包括JSON关键字
        if (MapUtils.getObject(ragConfig.getLlmConfig().getAdditional(), ProviderRequestService.KEY_RESPONSE_SCHEMA) != null && ragData.getRequest().isApi(ProviderRequest.REQUEST_OPENAI)) {
            WorkflowException.check(!(StringUtils.containsIgnoreCase(ragData.getPrompt(), "json")), "The request " + SplitUtils.join(ragData.getQuery()) + " must contain the `json` keyword when `rag_schema` is used: " + ragData.getPrompt(), ProtocolCode.C400);
        }
    }

    protected void checkSchemaOnRuntime(RagConfig ragConfig, RagData ragData) throws Exception {
        // 如果为Main且为Task上下文，Query必需包含JSON关键字
        if (SplitUtils.equals(ragData.getQuery(), MultiSourceNotifier.MAIN) && FeatureFlag.isTask(ragData.getQuery())) {
            WorkflowException.check(!(StringUtils.containsIgnoreCase(ragData.getPrompt(), "json")), "The request " + SplitUtils.join(ragData.getQuery()) + " must contain the `json` keyword: " + ragData.getQuery().getQuery(), ProtocolCode.C400);
        }
    }

    protected void checkFunCallPlaceholder(RagConfig ragConfig, RagData ragData) throws Exception {
        // 占位符未替换检查
        List<ProviderFunCall> providerFunCallL = ragData.getRequest().getFunCalls();
        if (!CollectionUtils.isEmpty(providerFunCallL)) {
            for (ProviderFunCall funCall : providerFunCallL) {
                WorkflowException.check(!(TemplateChecker.check(JsonUtils.write(funCall))), "The request " + SplitUtils.join(ragData.getQuery()) + "'s funcall must not contain placeholders: " + funCall.getName(), ProtocolCode.C400);
            }
        }
    }

    protected void checkPromptPlaceholder(RagConfig ragConfig, RagData ragData) throws Exception {
        // 占位符未替换检查
        WorkflowException.check(!(TemplateChecker.check(ragData.getQuery().getQuery())), "The request " + SplitUtils.join(ragData.getQuery()) + "'s query must not contain placeholders: " + ragData.getQuery().getQuery(), ProtocolCode.C400);
        WorkflowException.check(!(TemplateChecker.check(ragData.getPrompt())), "The request " + SplitUtils.join(ragData.getQuery()) + "'s prompt must not contain placeholders: " + ragData.getPrompt(), ProtocolCode.C400);
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Value("${debug:false}")
        protected Boolean debug;

        @Bean(RequestCheckRag.RAG_KEY)
        @ConditionalOnMissingBean(name = RequestCheckRag.RAG_KEY)
        public RequestCheckRag requestChecker() throws Exception {
            RequestCheckRag requestChecker = new RequestCheckRag();
            BeanUtils.copyProperties(this, requestChecker);
            log.info("RequestCheckRag inited, timeout4Condition={}", requestChecker.getTimeout4Condition());
            return requestChecker;
        }
    }
}
