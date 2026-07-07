package ai.deepright.llm.optimize;

import ai.deepright.feature.FeatureFlag;
import ai.deepright.llm.provider.RequestProviderUtils;
import ai.deepright.memory.MemoryService;
import ai.deepright.memory.impl.DefMemoryService;
import ai.deepright.plan.PlanUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCall;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.Iterator;

@Slf4j
@Getter
@Setter
// 从FunCall中删除未配置的功能
public class RequestFunCall {

    public static final String NAME = "request_funcall";

    protected MemoryService memoryService;

    protected NamesService namesService;

    public void allowed(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        if (!CollectionUtils.isEmpty(providerRequest.getFunCalls())) {
            Iterator<ProviderFunCall> funCallData = providerRequest.getFunCalls().iterator();
            while (funCallData.hasNext()) {
                ProviderFunCall funCall = funCallData.next();
                String funCallName = SplitUtils.join(this.namesService.decode(funCall.getName()));
                if (!this.isProviderSupport(providerRequest, llmConfig, llmQuery, funCallName)) {
                    funCallData.remove();
                    if (log.isInfoEnabled()) {
                        log.info("The fun call {} has been removed by `isProviderSupport`", funCallName);
                    }
                } else if (!this.isMemorySupport(providerRequest, llmConfig, llmQuery, funCallName)) {
                    funCallData.remove();
                    if (log.isDebugEnabled()) {
                        log.debug("The fun call {} has been removed by `isMemorySupport`", funCallName);
                    }
                } else if (!this.isOpenTeamMode(providerRequest, llmConfig, llmQuery, funCallName)) {
                    funCallData.remove();
                    if (log.isDebugEnabled()) {
                        log.debug("The fun call {} has been removed by `isOpenTeamMode`", funCallName);
                    }
                } else if (!this.shouldPlanCreate(providerRequest, llmConfig, llmQuery, funCallName)) {
                    funCallData.remove();
                    if (log.isInfoEnabled()) {
                        log.info("The fun call {} has been removed by `shouldPlanCreate`", funCallName);
                    }
                } else if (!this.shouldPlanUpdate(providerRequest, llmConfig, llmQuery, funCallName)) {
                    funCallData.remove();
                    if (log.isInfoEnabled()) {
                        log.info("The fun call {} has been removed by `shouldPlanUpdate`", funCallName);
                    }
                }
            }
        }
    }

    protected Boolean isProviderSupport(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery, String funcall) throws Exception {
        if ((StringUtils.equalsIgnoreCase(funcall, "media@file") || StringUtils.equalsIgnoreCase(funcall, "media@ocr"))) {
            return RequestProviderUtils.isMultiInputModel(providerRequest.getMessage());
        } else if (StringUtils.equalsIgnoreCase(funcall, "media@image")) {
            return RequestProviderUtils.isMultiOutputModel(providerRequest.getMessage());
        } else {
            return true;
        }
    }

    protected Boolean isMemorySupport(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery, String funcall) throws Exception {
        // memory@recall 并且 当前MemoryService支持
        if (StringUtils.equalsIgnoreCase(funcall, "memory@recall")) {
            return this.memoryService.support(providerRequest.getMessage());
        } else {
            return true;
        }
    }

    protected Boolean isOpenTeamMode(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery, String funcall) throws Exception {
        // Task且启动团队
        if (StringUtils.equalsIgnoreCase(funcall, "task@main")) {
            return FeatureFlag.isOpenTeamMode(llmQuery);
        } else {
            return true;
        }
    }

    protected Boolean shouldPlanCreate(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery, String funcall) throws Exception {
        if (StringUtils.equalsIgnoreCase(funcall, "plan@create")) {
            // 需要规划 且 没有规划
            return PlanUtils.shouldPlan(providerRequest.getMessage()) && StringUtils.isEmpty(PlanUtils.fetchPlan(providerRequest.getMessage()));
        } else {
            return true;
        }
    }

    protected Boolean shouldPlanUpdate(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery, String funcall) throws Exception {
        if (StringUtils.equalsIgnoreCase(funcall, "plan@update") || StringUtils.equalsIgnoreCase(funcall, "plan@delete")) {
            // 有规划
            return !StringUtils.isEmpty(PlanUtils.fetchPlan(providerRequest.getMessage()));
        } else {
            return true;
        }
    }

    @Configuration
    @Getter
    @Setter
    public static class InitConfig {

        @Autowired
        @Qualifier(DefMemoryService.NAME)
        protected MemoryService memoryService;

        @Autowired
        protected NamesService namesService;

        @Bean(RequestFunCall.NAME)
        @ConditionalOnMissingBean(name = RequestFunCall.NAME)
        public RequestFunCall requestFunCall() throws Exception {
            RequestFunCall requestFunCall = new RequestFunCall();
            BeanUtils.copyProperties(this, requestFunCall);
            log.info("RequestFunCall inited");
            return requestFunCall;
        }
    }
}
