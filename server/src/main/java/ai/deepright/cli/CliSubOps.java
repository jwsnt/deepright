package ai.deepright.cli;

import static org.springframework.util.ObjectUtils.isEmpty;

import ai.deepright.feature.FeatureFlag;
import ai.deepright.feature.FeatureUtils;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.media.MediaTransferUtils;
import lombok.*;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.ArrayList;
import java.util.List;

@Getter
@Setter
@Builder
@NoArgsConstructor
@AllArgsConstructor
// 软操作限制
public class CliSubOps {

    @Builder.Default
    protected Boolean exempted = false;

    @Builder.Default
    protected Boolean echo = true;

    protected List<String> app;

    protected List<String> w;

    protected List<String> r;

    // 豁免
    public Boolean isExempted() {
        return this.exempted;
    }

    public Boolean hasApp() {
        return !CollectionUtils.isEmpty(this.app);
    }

    public Boolean hasW() {
        return !CollectionUtils.isEmpty(this.w);
    }

    public Boolean hasR() {
        return !CollectionUtils.isEmpty(this.r);
    }

    public String getApps() throws Exception {
        return !CollectionUtils.isEmpty(this.app) ? StringUtils.join(this.app, ",") : "";
    }

    public CliSubOps reConfig(WorkflowTask workTask) throws Exception {
        if (!CollectionUtils.isEmpty(this.w)) {
            this.w = this.configPath(workTask, this.w);
        }
        if (!CollectionUtils.isEmpty(this.r)) {
            this.r = this.configPath(workTask, this.r);
        }
        return this;
    }

    // 指定命令（如cat）需要过滤掉前缀file:///，且为绝对路径。其他指令不需要转换
    protected List<String> configPath(WorkflowTask workTask, List<String> path) throws Exception {
        List<String> configPath = new ArrayList<String>(path.size());
        for (String each : path) {
            // Echo Cat 前缀容错
            if (!MediaTransferUtils.isNetwork(each) && !CollectionUtils.isEmpty(this.app) && (this.app.contains("echo") || this.app.contains("cat"))) {
                // 去除头尾转义
                String escape = FeatureUtils.escapeFile(workTask, each).replaceAll("^'|'$", "");
                if (!FeatureFlag.isAbsolutePath(workTask, escape)) {
                    throw new WorkflowException(each + " must be an absolute path").needSilent();
                }
                configPath.add(escape);
            } else {
                configPath.add(each);
            }
        }
        return configPath;
    }
}
