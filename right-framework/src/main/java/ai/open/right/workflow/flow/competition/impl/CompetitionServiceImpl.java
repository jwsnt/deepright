package ai.open.right.workflow.flow.competition.impl;

import ai.open.right.workflow.condition.Condition;
import ai.open.right.workflow.condition.ConditionUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.competition.CompetitionConfig;
import ai.open.right.workflow.flow.competition.CompetitionService;
import ai.open.right.workflow.flow.competition.ConditionConfig;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.ArrayList;
import java.util.List;

@Slf4j
@Setter
@Getter
// 多路竞争思考链（Workflow）选择一个
public class CompetitionServiceImpl implements CompetitionService {

    protected NotifierService notifierService;

    // 调用下游思考链（Workflow）的超时
    protected Integer timeout;

    @Override
    public String compete(CompetitionConfig competitionConfig, WorkflowTask workTask) throws Exception {
        Assert.isTrue(competitionConfig.hasConditions(), "Competition can not be empty, please check config");
        List<ConditionTask> conditionTasks = new ArrayList<ConditionTask>();
        for (ConditionConfig conditionConfig : competitionConfig.getConditionConfigs()) {
            Assert.hasText(conditionConfig.getDynamic(), "Competition configuration can not find `dynamic`: " + conditionConfig);
            SyncConfig syncConfig = SyncConfig.builder()
                    .timeout(competitionConfig.getTimeout(this.timeout))
                    .workflow(conditionConfig.getCondition())
                    .workTask(workTask)
                    .build();
            SyncWorkflowTask syncWorkflowTask = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig);
            conditionTasks.add(ConditionTask.builder()
                    .syncWorkflowTask(syncWorkflowTask)
                    .conditionConfig(conditionConfig)
                    .build());
        }
        for (ConditionTask each : conditionTasks) {
            try {
                // 逐个顺序检查，如果符合则立即返回
                if (this.checkCondition(each).print().getCondition()) {
                    return each.getConditionConfig().getDynamic();
                }
            } catch (Exception e) {
                // 任一失败是否终止
                if (competitionConfig.getStopOnFailed()) {
                    throw e;
                } else {
                    if (log.isInfoEnabled()) {
                        log.info(e.getMessage(), e);
                    }
                }
            }
        }
        Assert.isTrue(competitionConfig.hasTarget(), "Default target can not be empty");
        if (log.isInfoEnabled()) {
            log.info("Using default target={}", competitionConfig.getDynamic());
        }
        return competitionConfig.getDynamic();
    }

    protected Condition checkCondition(ConditionTask task) throws Exception {
        // True: True/true/Yes/Y/1
        // False: False/false/No/N/0 and Other
        // Json: {...,"condition":true/false/0/1}
        String condition = StringUtils.lowerCase(task.getSyncWorkflowTask().get());
        return ConditionUtils.checkCondition(condition);
    }

    @Builder
    @Setter
    @Getter
    public static class ConditionTask {

        protected SyncWorkflowTask syncWorkflowTask;

        protected ConditionConfig conditionConfig;
    }

    @ConditionalOnProperty(name = "competition.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NotifierService notifierService;

        @Value("${competition.timeout:1800000}")
        // 调用下游思考链（Workflow）的超时
        protected Integer timeout;

        @Bean
        @ConditionalOnMissingBean(value = CompetitionService.class)
        public CompetitionService competitionService() throws Exception {
            CompetitionServiceImpl competitionService = new CompetitionServiceImpl();
            BeanUtils.copyProperties(this, competitionService);
            log.info("CompetitionServiceImpl inited: timeout={}", competitionService.getTimeout());
            return competitionService;
        }
    }
}
