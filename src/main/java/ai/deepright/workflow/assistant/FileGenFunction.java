package ai.deepright.workflow.assistant;

import ai.deepright.cli.CliPubData;
import ai.deepright.cli.CliPubSub;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.cli.CliSubOps;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.router.RouterDevice;
import ai.open.right.WorkflowException;
import ai.open.right.utils.BytesUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SuffixUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.file.impl.SysStore;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.function.impl.BaseFunction;
import ai.open.right.workflow.sync.SyncConfig;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.List;
import java.util.Map;

@Slf4j
@Getter
@Setter
// 文件创建
public class FileGenFunction extends BaseFunction {

    public static final String LANG_KEY_ASSISTANT_FILEGEN_HINT = "assistant.filegen.hint";

    public static final String NAME = "fun_file_gen";

    protected CloseableHttpAsyncClient resource;

    protected CliSubFetcher cliSubFetcher;

    protected SysStore sysStore;

    // 大于此值（字节，默认1.5M）的文件会被上传到DefStore后下发URL
    protected Integer oversize;

    // 请求Refer文件的最大大小（防止LLM过载，默认25M）
    protected Integer maxsize;

    // 毫秒
    protected Integer timeout;

    public Object call(FunctionContext functionContext) throws Exception {
        WorkflowTask workTask = functionContext.getWorkTask().printQuery();
        Map<String, Object> query = this.buildQuery(workTask, JsonUtils.read(workTask.getQuery(), Map.class));
        // 调用实际Workflow
        SyncConfig syncConfig = SyncConfig.builder()
                // 从media.json获取Workflow节点配置（实际的分析节点）
                .workflow(MapUtils.getString(functionContext.getFunctionConfig().getEnvironment(), "workflow"))
                .reQuery(JsonUtils.write(query))
                .timeout(this.timeout)
                .workTask(workTask)
                .build();
        String response = this.localhost(functionContext, syncConfig).get();
        // 获取结果
        return this.buildResult(workTask, JsonUtils.read(response, Map.class), String.class.cast(query.get("file_path")));
    }

    // 将创建文件写入CLI
    protected Map<String, Object> buildResult(WorkflowTask workTask, Map<String, Object> data, String file) throws Exception {
        String content = String.class.cast(data.remove("content"));
        String why = String.class.cast(data.get("why_do_this"));
        // 禁止exempted=true豁免
        CliPubData pubData = this.cliSubFetcher.command(workTask, new RouterDevice(workTask), CliSubOps.builder()
                .w(List.of(file))
                .build(), CliPubSub.buildPushCmd(workTask, this.sysStore, this.oversize, content, file), why);
        WorkflowException.checkCondition(!(pubData.isOk()), pubData.getCmd());
        data.put("file_path", file);
        return data;
    }

    // 提取Refer_uris内容
    protected Map<String, Object> buildQuery(WorkflowTask workTask, Map<String, Object> query) throws Exception {
        List<String> refers = List.class.cast(query.remove("refer_uris"));
        // 异常需要抛给模型
        WorkflowException.checkCondition(CollectionUtils.isEmpty(refers), "The refer_uris can not be empty");
        for (String refer : refers) {
            WorkflowException.checkCondition(StringUtils.isEmpty(refer), "The refer_uris cannot contain empty values");
        }
        if (!refers.isEmpty()) {
            // 禁止exempted=true豁免
            CliPubData pubData = this.cliSubFetcher.fetch(workTask, new RouterDevice(workTask), CliSubOps.builder()
                    .r(refers)
                    .build(), refers, "");
            WorkflowException.checkCondition(SuffixUtils.isBinary(pubData.getSuffix()), "The referenced file cannot be a binary file: " + pubData.getSuffix());
            WorkflowException.checkCondition(!pubData.isOk(), pubData.getCmd());
            if (!StringUtils.isEmpty(pubData.getCmd())) {
                // 强制转换为TEXT
                String referBody = pubData.forceText(this.resource, this.sysStore, this.timeout).getCmd();
                this.checkSize(workTask, query, BytesUtils.utf8Bytes(referBody));
                query.put("refer_body", referBody);
            }
        }
        return query;
    }

    protected void checkSize(WorkflowTask workTask, Map<String, Object> query, Integer size) throws Exception {
        // 提交LLM前检查（提示小于min，大于max），但仅检查大于，防止上下文溢出
        // MinSize仅用于LLM提示
        if (size > this.maxsize) {
            throw new WorkflowException(XmlResourceLang.get(FileGenFunction.LANG_KEY_ASSISTANT_FILEGEN_HINT)
                    .replace("#max", String.valueOf(this.maxsize)))
                    .needSilent();
        }
    }

    @Configuration
    @Getter
    @Setter
    public static class InitConfig {

        @Autowired
        protected CloseableHttpAsyncClient resource;

        @Autowired
        protected CliSubFetcher cliSubFetcher;

        @Autowired
        protected SysStore sysStore;

        @Value("${cli.push.oversize:1048576}")
        protected Integer oversize;

        // 与配置文件共享
        @Value("${file_gen.maxsize:26214400}")
        protected Integer maxsize;

        @Value("${file_gen.timeout:10000}")
        protected Integer timeout;

        @Bean(FileGenFunction.NAME)
        @ConditionalOnMissingBean(name = FileGenFunction.NAME)
        public FileGenFunction fileGenFunction() throws Exception {
            FileGenFunction fileGenFunction = new FileGenFunction();
            BeanUtils.copyProperties(this, fileGenFunction);
            log.info("FileGenFunction inited");
            return fileGenFunction;
        }
    }
}
