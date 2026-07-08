package ai.deepright.task;

import static org.springframework.util.ObjectUtils.isEmpty;

import static org.springframework.util.StringUtils.hasText;




import ai.open.right.protocol.ProtocolCode;

import ai.open.right.WorkflowException;

import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.llm.notifier.MultiSourceNotifier;
import ai.deepright.utils.TemplateChecker;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.util.Map;

@Slf4j
@Getter
@Setter
public class TaskRag extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_output";

    protected ResourceService resourceService;

    protected String template4schemaPlugin;

    protected String template4schemaHtml;

    protected String template4schemaJson;

    protected String template4schemaDef;

    @PostConstruct
    public void init() throws Exception {
        // 由IOUtils关闭资源
        this.template4schemaPlugin = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4schemaPlugin).openStream()), StandardCharsets.UTF_8);
        this.template4schemaHtml = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4schemaHtml).openStream()), StandardCharsets.UTF_8);
        this.template4schemaJson = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4schemaJson).openStream()), StandardCharsets.UTF_8);
        this.template4schemaDef = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4schemaDef).openStream()), StandardCharsets.UTF_8);
        WorkflowException.check(!hasText(this.template4schemaJson), "The template json schema must not be empty", ProtocolCode.C400);
        WorkflowException.check(!hasText(this.template4schemaHtml), "The template html schema must not be empty", ProtocolCode.C400);
        WorkflowException.check(!hasText(this.template4schemaDef), "The template def schema must not be empty", ProtocolCode.C400);
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        String schema = this.buildSchema(ragConfig, ragData);
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), schema);
        return new RagAtOnce(ragConfig);
    }

    protected String buildSchema(RagConfig ragConfig, RagData ragData) throws Exception {
        if (SplitUtils.equals(ragData.getQuery(), MultiSourceNotifier.MAIN) && (FeatureFlag.isTask(ragData.getQuery()) || FeatureFlag.isResponseSchema(ragData.getQuery()))) {
            String responseSchema = null;
            // Task first
            if (FeatureFlag.isTask(ragData.getQuery())) {
                // @see CliTaskFunction metadata
                Map<String, Object> output = ragData.getQuery().getMetadata(FeatureField.KEY_OUTPUT, Map.class);
                WorkflowException.check(isEmpty(output), "The output can not be empty, please check the task.json", ProtocolCode.C400);
                responseSchema = JsonUtils.write(output);
            } else if (FeatureFlag.isResponseSchema(ragData.getQuery())) {
                responseSchema = this.buildResponseSchema(ragData.getQuery());
            }
            // 精确匹配
            WorkflowException.check(!hasText(responseSchema), "The schema template can not be empty", ProtocolCode.C400);
            String schema = this.buildPlugin(ragData, this.template4schemaJson.replace("#schema", StringUtils.defaultIfEmpty(responseSchema, "")));
            if (log.isWarnEnabled() && !TemplateChecker.check(schema)) {
                log.warn("The schema template contains unexpected characters, please check: {}", schema);
            }
            return schema;
        } else {
            return this.buildTemplate(ragConfig, ragData);
        }
    }

    protected String buildTemplate(RagConfig ragConfig, RagData ragData) throws Exception {
        return FeatureFlag.isHtml(ragData.getQuery()) ? this.template4schemaHtml : this.template4schemaDef;
    }

    protected String buildPlugin(RagData ragData, String responseSchema) throws Exception {
        if (!StringUtils.isEmpty(responseSchema)) {
            String cronType = this.buildCronType(ragData.getQuery());
            responseSchema = responseSchema.replace("#plugin", !StringUtils.isEmpty(cronType) ? this.template4schemaPlugin.replace("#plugin", cronType) : "");
            if (log.isWarnEnabled() && !TemplateChecker.check(responseSchema)) {
                log.warn("The response schema template contains unexpected characters; please check: {}", responseSchema);
            }
        }
        return responseSchema;
    }

    protected String buildResponseSchema(WorkflowTask workTask) throws Exception {
        return MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_RESPONSE_SCHEMA);
    }

    protected String buildCronType(WorkflowTask workTask) throws Exception {
        return MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_CRON_TYPE);
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Value("${task.rag.schema.html:classpath:config/task/schema_plugin.md}")
        protected String template4schemaPlugin;

        @Value("${task.rag.schema.html:classpath:config/task/schema_html.md}")
        protected String template4schemaHtml;

        @Value("${task.rag.schema.json:classpath:config/task/schema_json.md}")
        protected String template4schemaJson;

        @Value("${task.rag.schema.def:classpath:config/task/schema_def.md}")
        protected String template4schemaDef;

        @Bean(TaskRag.RAG_KEY)
        @ConditionalOnMissingBean(name = TaskRag.RAG_KEY)
        public TaskRag taskRag() throws Exception {
            TaskRag taskRag = new TaskRag();
            BeanUtils.copyProperties(this, taskRag);
            log.info("CliTaskRag inited, timeout4Condition={}", taskRag.getTimeout4Condition());
            return taskRag;
        }
    }
}
