package ai.deepright.router;

import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.collections.MapUtils;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Getter
@Setter
public class RouterAgent {

    // 多模态配置(Provider,Config)
    protected Map<String, Map<String, Object>> media;

    private List<Map<String, Object>> skills;

    protected String description;

    protected String knowledge;

    protected String workspace;

    // 是否为思考模式
    protected Boolean thinking;

    protected String provider;

    @JsonProperty("router_disable")
    // 只影响是否出现在router列表里，不阻止当前Device路由
    protected Boolean disable = true;

    @JsonProperty("agentId")
    protected String agent;

    protected String soul;

    protected String user;

    // Cli@Get构造，没有Rag，需要手动Set
    public RouterDevice buildRouterDevice(WorkflowTask workTask) throws Exception {
        // WorkflowTask workTask, String router, String device, String agent, String desc
        RouterDevice routerDevice = new RouterDevice(workTask, workTask.getDevice(), this.agent);
        Map<String, Object> metadata = new HashMap<String, Object>();
        // WorkTask的Metadata有状态位，不能全部复制
        metadata.put(FeatureField.KEY_ROUTER_DESC, MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_ROUTER_DESC));
        metadata.put(FeatureField.KEY_TERMINAL, FeatureUtils.buildTerminal(workTask));
        metadata.put(FeatureField.KEY_TIMEZONE, FeatureUtils.buildTimezone(workTask));
        metadata.put(FeatureField.KEY_GATEWAY, FeatureUtils.buildGateway(workTask));
        metadata.put(FeatureField.KEY_SYS, FeatureUtils.buildSys(workTask));
        metadata.put(FeatureField.KEY_APP, FeatureUtils.buildApp(workTask));
        metadata.put(FeatureField.KEY_KNOWLEDGE_CONTENT, this.knowledge);
        metadata.put(FeatureField.KEY_WORKSPACE, this.workspace);
        metadata.put(FeatureField.KEY_THINKING, this.thinking);
        metadata.put(FeatureField.KEY_AGENTID, this.agent);
        metadata.put(FeatureField.KEY_SKILLS, this.skills);
        metadata.put(FeatureField.KEY_MEDIA, this.media);
        metadata.put(FeatureField.KEY_SOUL, this.soul);
        metadata.put(FeatureField.KEY_USER, this.user);
        routerDevice.setGateway(FeatureUtils.buildGateway(workTask));
        routerDevice.setDescription(this.description);
        routerDevice.setWorkspace(this.workspace);
        routerDevice.setProvider(this.provider);
        routerDevice.setEnabled(!this.disable);
        routerDevice.setMetadata(metadata);
        return routerDevice;
    }
}
