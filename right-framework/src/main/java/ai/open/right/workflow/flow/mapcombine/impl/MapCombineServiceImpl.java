package ai.open.right.workflow.flow.mapcombine.impl;

import ai.open.right.WorkflowException;
import ai.open.right.utils.CollectionsUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.mapcombine.MapCombineConfig;
import ai.open.right.workflow.flow.mapcombine.MapCombineService;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import ai.open.right.workflow.sync.impl.NotifierCallable;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.StringReader;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.stream.Collectors;

@Slf4j
@Setter
@Getter
public class MapCombineServiceImpl implements MapCombineService {

    protected NotifierService notifierService;

    // MapCombine调用下游思考链（Workflow）超时
    protected Integer timeout4Llm;

    // MapCombine最大分片数量
    protected Integer segment;

    @Override
    public String execute(MapCombineConfig mapCombineConfig, WorkflowTask workTask) throws Exception {
        // 包含Mapping或Combine的思考链配置，且Combine批次必须要大于0
        Assert.isTrue(mapCombineConfig.isValid(), "The MapCombine must include the configuration for either Mapping or Combine, and the Combine batch size must be greater than 0, please check config");
        SyncConfig syncSplit = SyncConfig.builder()
                .timeout(mapCombineConfig.getTimeout4Llm(this.timeout4Llm))
                .workflow(mapCombineConfig.getMapping().getSplit())
                .reQuery(workTask.getQuery())
                .workTask(workTask)
                .build();
        String splitResponse = SyncWorkflowTask.exeWorkflow(this.notifierService, syncSplit).get();
        Assert.hasText(splitResponse, "Split response can not be empty");
        // 拆分为Segment
        List<String> segments = this.split(splitResponse);
        if (log.isDebugEnabled()) {
            log.debug("MapCombine split={}", segments);
        }
        List<SyncWorkflowTask> mapWorkflowTasks = new ArrayList<SyncWorkflowTask>();
        Assert.isTrue(this.segment > segments.size(), "Segments must be less than the config: " + this.segment);
        for (String segment : segments) {
            if (!StringUtils.isEmpty(segment.trim())) {
                SyncConfig syncDynamic = SyncConfig.builder()
                        // 自定通知方式（Localhost/Endpoint/Source）
                        .syncCallable(mapCombineConfig.getMapping().hasNotifier() ? new NotifierCallable(mapCombineConfig.getMapping().getNotifier()) : null)
                        .timeout(mapCombineConfig.getTimeout4Llm(this.timeout4Llm))
                        .workflow(mapCombineConfig.getMapping().getDynamic())
                        .workTask(workTask)
                        .reQuery(segment)
                        .build();
                mapWorkflowTasks.add(SyncWorkflowTask.exeWorkflow(this.notifierService, syncDynamic));
            }
        }
        List<String> mapResponses = this.getMapResponse(mapCombineConfig, mapWorkflowTasks);
        if (log.isDebugEnabled()) {
            log.debug("MapCombine segment map response={}", mapResponses);
        }
        return this.getCombineResponse(mapCombineConfig, workTask, mapResponses);
    }

    protected String getCombineResponse(MapCombineConfig mapCombineConfig, WorkflowTask workTask, List<String> mapResponses) throws Exception {
        List<String> mapBatchResponses = new ArrayList<String>();
        // Split Collection，将一个大的列表（List）按照指定的大小拆分成多个子列表
        for (List<String> eachBatch : CollectionsUtils.partition(mapResponses, mapCombineConfig.getCombine().getBatch())) {
            mapBatchResponses.add(eachBatch.stream().collect(Collectors.joining(System.lineSeparator())));
        }
        if (log.isInfoEnabled()) {
            log.info("MapCombine segment map batch responses={}", mapBatchResponses);
        }
        StringBuffer combineResponse = new StringBuffer();
        for (String mapBatchResponse : mapBatchResponses) {
            try {
                String request = StringUtils.trim(combineResponse + System.lineSeparator() + mapBatchResponse);
                SyncConfig syncConfig = SyncConfig.builder()
                        // 指定通知方式
                        .syncCallable(mapCombineConfig.getCombine().hasNotifier() ? new NotifierCallable(mapCombineConfig.getCombine().getNotifier()) : null)
                        .timeout(mapCombineConfig.getTimeout4Llm(this.timeout4Llm))
                        .workflow(mapCombineConfig.getCombine().getDynamic())
                        .workTask(workTask)
                        .reQuery(request)
                        .build();
                String response = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
                if (log.isDebugEnabled()) {
                    log.debug("MapCombine request={} and response={}", request, response);
                }
                Assert.hasText(response, "MapCombine response can not be empty: " + request);
                combineResponse.append(response);
            } catch (Exception e) {
                if (!mapCombineConfig.getCombine().getStopOnFailed()) {
                    WorkflowException.dolog(e);
                } else {
                    throw e;
                }
            }
        }
        String response = combineResponse.toString();
        if (log.isInfoEnabled()) {
            log.info("MapCombine's combine: response={}", response);
        }
        Assert.hasText(response, "Response can not be empty");
        return response;
    }

    protected List<String> getMapResponse(MapCombineConfig mapCombineConfig, List<SyncWorkflowTask> syncWorkflowTasks) throws Exception {
        List<String> mapResponses = new ArrayList<String>();
        for (SyncWorkflowTask syncWorkflowTask : syncWorkflowTasks) {
            try {
                String mapResponse = syncWorkflowTask.get();
                Assert.hasText(mapResponse, "Map response can not be empty");
                mapResponses.add(mapResponse);
            } catch (Exception e) {
                if (!mapCombineConfig.getMapping().getStopOnFailed()) {
                    WorkflowException.dolog(e);
                } else {
                    throw e;
                }
            }
        }
        if (log.isInfoEnabled()) {
            log.info("MapCombine's map={}", mapResponses);
        }
        return mapResponses;
    }

    public List<String> split(String content) throws Exception {
        if (JsonUtils.like(content)) {
            String[] segment = JsonUtils.read(content, String[].class);
            List<String> split = Arrays.asList(segment);
            if (log.isDebugEnabled()) {
                log.debug("MapCombine split content after using json={}", split);
            }
            return split;
        } else {
            try (StringReader reader = new StringReader(content)) {
                List<String> split = IOUtils.readLines(reader);
                if (log.isDebugEnabled()) {
                    log.debug("MapCombine split content after reading line={}", split);
                }
                return split;
            }
        }
    }

    @ConditionalOnProperty(name = "mapCombine.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NotifierService notifierService;

        @Value("${mapcombine.timeout.llm:1800000}")
        // MapCombine调用下游思考链（Workflow）超时
        protected Integer timeout4Llm;

        @Value("${mapcombine.segment:10}")
        // MapCombine最大分片数量
        protected Integer segment;

        @Bean
        @ConditionalOnMissingBean(value = MapCombineService.class)
        public MapCombineService mapCombineService() throws Exception {
            MapCombineServiceImpl mapCombineService = new MapCombineServiceImpl();
            BeanUtils.copyProperties(this, mapCombineService);
            log.info("MapCombineServiceImpl inited: timeout4Llm={},segment={}", mapCombineService.getTimeout4Llm(), mapCombineService.getSegment());
            return mapCombineService;
        }
    }
}
