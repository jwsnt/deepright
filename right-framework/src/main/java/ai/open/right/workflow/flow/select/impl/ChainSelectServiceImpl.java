package ai.open.right.workflow.flow.select.impl;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.FunctionConfig;
import ai.open.right.workflow.flow.function.FunctionService;
import ai.open.right.workflow.flow.select.ChainSelectConfig;
import ai.open.right.workflow.flow.select.ChainSelectService;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

@Setter
@Getter
@Slf4j
public class ChainSelectServiceImpl implements ChainSelectService {

    protected FunctionService functionService;

    protected NotifierService notifierService;

    // 摘要调用下游思考链（Workflow）的超时
    protected Integer timeout4Llm;

    @Override
    public String selectChain(ChainSelectConfig chainSelectConfig, WorkflowTask workTask) throws Exception {
        try {
            if (chainSelectConfig.hasFunction()) {
                // 使用Function获取Chain
                return this.buildChainFunction(chainSelectConfig, workTask);
            } else if (chainSelectConfig.hasDynamic()) {
                // 使用思考链（Workflow）获取Chain
                return this.buildChainDynamic(chainSelectConfig, workTask);
            }
            log.warn("ChainSelectServiceImpl will use default chain={}", chainSelectConfig.getChain());
            return chainSelectConfig.getChain();
        } catch (Exception e) {
            if (chainSelectConfig.hasChain()) {
                if (log.isDebugEnabled()) {
                    log.debug(e.getMessage(), e);
                }
                return chainSelectConfig.getChain();
            } else {
                throw e;
            }
        }
    }

    protected String buildChainDynamic(ChainSelectConfig chainSelectConfig, WorkflowTask workTask) throws Exception {
        SyncConfig syncConfig = SyncConfig.builder()
                .timeout(chainSelectConfig.getTimeout(this.timeout4Llm))
                .workflow(chainSelectConfig.getDynamic())
                .reQuery(workTask.getQuery())
                .workTask(workTask)
                .build();
        String chain = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
        Assert.hasText(chain, "ChainSelectServiceImpl can not select empty `chain`");
        return chain;
    }

    protected String buildChainFunction(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
        Assert.notNull(this.functionService, "The functionService can not be empty, please config `function.enable`");
        Object chain = this.functionService.call(functionConfig, workTask);
        Assert.notNull(chain, "ChainSelectServiceImpl can not select empty `chain`");
        return chain.toString();
    }

    @ConditionalOnProperty(name = "chainSelector.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected FunctionService functionService;

        @Autowired
        protected NotifierService notifierService;

        @Value("${chainSelector.timeout.llm:1800000}")
        // 摘要调用下游思考链（Workflow）的超时
        protected Integer timeout4Llm;

        @Bean
        @ConditionalOnMissingBean(value = ChainSelectService.class)
        public ChainSelectService chainSelectService() throws Exception {
            ChainSelectServiceImpl chainSelectService = new ChainSelectServiceImpl();
            BeanUtils.copyProperties(this, chainSelectService);
            log.info("ChainSelectServiceImpl inited, timeout4Llm={}", chainSelectService.getTimeout4Llm());
            return chainSelectService;
        }
    }
}
