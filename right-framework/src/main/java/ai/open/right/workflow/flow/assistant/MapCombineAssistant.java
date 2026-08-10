package ai.open.right.workflow.flow.assistant;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.mapcombine.MapCombineService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

@Slf4j
@Setter
@Getter
// Map Combine（拆分，归纳）
public class MapCombineAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-mapCombine";

    protected MapCombineService mapCombineService;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Assert.isTrue(workflowConfig.hasMapCombine(), "MapCombine config can not be empty, please check config");
        String query = this.mapCombineService.execute(workflowConfig.getMapCombineConfig(), workTask);
        this.chainOr2Endpoint(workflowConfig, workTask, query);
    }

    @ConditionalOnProperty(name = "mapCombine.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected MapCombineService mapCombineService;

        @Bean(MapCombineAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = MapCombineAssistant.WORKFLOW_NAME)
        public MapCombineAssistant mapCombineAssistant() throws Exception {
            MapCombineAssistant mapCombineAssistant = new MapCombineAssistant();
            BeanUtils.copyProperties(this, mapCombineAssistant);
            log.info("MapCombineAssistant inited");
            return mapCombineAssistant;
        }
    }
}
