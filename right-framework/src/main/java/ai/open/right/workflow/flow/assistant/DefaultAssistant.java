package ai.open.right.workflow.flow.assistant;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.Protocol;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.notify.NotifierService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.Map;

@Slf4j
@Setter
@Getter
public class DefaultAssistant extends ChainAssistant implements Assistant {

    public static final String WORKFLOW_NAME = "def";

    protected Map<String, LLMQueryService> llmQueryService;

    protected SignalFactory signalFactory;

    @Override
    // 执行前配置WorkTask，可用于继承类配置
    public void config(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        try {
            // 1,请求来自FunCall，且配置了拆箱属性，且为JSON Map
            // 2,请求声明JSON
            if (JsonUtils.map(workTask.getQuery()) && (this.isFunCallFetch(workflowConfig, workTask) || this.isRequestJson(workflowConfig, workTask))) {
                // 转换为Map后使用Unbox提取
                Object reQuery = MapUtils.getObject(workTask.getObjectQuery(Map.class), workflowConfig.getUnboxed());
                if (reQuery == null) {
                    if (log.isDebugEnabled()) {
                        log.debug("Unboxed query can not be empty, query={}", workTask.getQuery());
                    }
                } else {
                    workTask.setQuery(JsonUtils.write(reQuery));
                }
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    protected Boolean isFunCallFetch(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        LLMFunCallRequest funCall = workTask.getMetadata(ProviderRequestService.KEY_FUN_FETCH, LLMFunCallRequest.class);
        return funCall != null && !StringUtils.isEmpty(funCall.getName()) && workflowConfig.hasFunCall();
    }

    protected Boolean isRequestJson(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        return StringUtils.equalsIgnoreCase(workflowConfig.getRequest(), WorkflowConfig.REQUEST_JSON);
    }

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Assert.isTrue(Protocol.CHAT.equalsIgnoreCase(workTask.getProtocol()), "Protocol is not chat, please check query");
        if (!this.allowed(workTask)) {
            if (log.isInfoEnabled()) {
                log.info("Reject task={} ", workTask.getDimension());
            }
            return;
        }
        LLMConfig llmConfig = workflowConfig.getLlmConfig();
        LLMQueryService llmQueryService = this.llmQueryService.get(llmConfig.getProvider());
        Assert.notNull(llmQueryService, "Can not find provier: " + llmConfig.getProvider());
        llmQueryService.query(LLMQuery.build(workTask), llmConfig, this.signalFactory != null ? this.signalFactory.signal(workflowConfig) : null);
    }

    // 拒绝策略，子类重写
    public Boolean allowed(WorkflowTask workTask) {
        return true;
    }
    @Setter
    @Getter
    public static class DefInitConfig extends ChainInitConfig {

        @Autowired
        protected Map<String, LLMQueryService> llmQueryService;

        @Autowired
        protected NotifierService notifierService;

        @Autowired(required = false)
        protected SignalFactory signalFactory;
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Bean(DefaultAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = DefaultAssistant.WORKFLOW_NAME)
        public DefaultAssistant defaultAssistant() throws Exception {
            DefaultAssistant defaultAssistant = new DefaultAssistant();
            BeanUtils.copyProperties(this, defaultAssistant);
            log.info("DefaultAssistant inited");
            return defaultAssistant;
        }
    }
}
