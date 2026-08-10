package ai.open.right.workflow.flow.llm.provider.seedream;

import ai.open.right.workflow.flow.llm.provider.ProviderImageConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.HashMap;
import java.util.Map;

@Setter
@Getter
public class SeedreamRequest extends ProviderRequest implements ProviderImageConfig {

    public static final DefaultMedia EMPTY = new DefaultMedia();

    protected Map<String, Object> sequentialOptions;

    protected Map<String, Object> optimizeOptions;

    protected SeedreamMedia seedMedia = SeedreamRequest.EMPTY;

    protected String sequential;

    protected Double guidance;

    protected String mimeType;

    protected Integer images;

    protected String format;

    protected Integer seed;

    protected String model;

    protected String size;

    @Override
    public void setImageConfig(Map<String, Object> imageConfig) {
        if (MapUtils.isEmpty(imageConfig)) {
            return;
        }
        Map<String, Object> sequentialOptions = MapUtils.getMap(imageConfig, "sequentialOptions");
        Map<String, Object> optimizeOptions = MapUtils.getMap(imageConfig, "optimizeOptions");
        String sequential = MapUtils.getString(imageConfig, "sequential");
        Double guidance = MapUtils.getDouble(imageConfig, "guidance");
        String mimeType = MapUtils.getString(imageConfig, "mimeType");
        Integer images = MapUtils.getInteger(imageConfig, "images");
        String format = MapUtils.getString(imageConfig, "format");
        Integer seed = MapUtils.getInteger(imageConfig, "seed");
        String model = MapUtils.getString(imageConfig, "model");
        String size = MapUtils.getString(imageConfig, "size");
        if (!MapUtils.isEmpty(sequentialOptions)) {
            this.sequentialOptions = sequentialOptions;
        }
        if (!MapUtils.isEmpty(optimizeOptions)) {
            this.optimizeOptions = optimizeOptions;
        }
        this.sequential = !StringUtils.isEmpty(sequential) ? sequential : this.sequential;
        this.mimeType = !StringUtils.isEmpty(mimeType) ? mimeType : this.mimeType;
        this.format = !StringUtils.isEmpty(format) ? format : this.format;
        this.model = !StringUtils.isEmpty(model) ? model : this.model;
        this.guidance = guidance != null ? guidance : this.guidance;
        this.size = !StringUtils.isEmpty(size) ? size : this.size;
        this.images = images != null ? images : this.images;
        this.seed = seed != null ? seed : this.seed;
    }

    @Override
    public Map<String, Object> getImageConfig() {
        Map<String, Object> imageConfig = new HashMap<String, Object>();
        if (!MapUtils.isEmpty(this.sequentialOptions)) {
            imageConfig.put("sequentialOptions", this.sequentialOptions);
        }
        if (!MapUtils.isEmpty(this.optimizeOptions)) {
            imageConfig.put("optimizeOptions", this.optimizeOptions);
        }
        if (!StringUtils.isEmpty(this.sequential)) {
            imageConfig.put("sequential", this.sequential);
        }
        if (this.guidance != null) {
            imageConfig.put("guidance", this.guidance);
        }
        if (!StringUtils.isEmpty(this.mimeType)) {
            imageConfig.put("mimeType", this.mimeType);
        }
        if (this.images != null) {
            imageConfig.put("images", this.images);
        }
        if (!StringUtils.isEmpty(this.format)) {
            imageConfig.put("format", this.format);
        }
        if (!StringUtils.isEmpty(this.model)) {
            imageConfig.put("model", this.model);
        }
        if (!StringUtils.isEmpty(this.size)) {
            imageConfig.put("size", this.size);
        }
        if (this.seed != null) {
            imageConfig.put("seed", this.seed);
        }
        return imageConfig;
    }

    public static class DefaultMedia implements SeedreamMedia {

        @Override
        public String getPrefix(String type) {
            return "data:" + type + ";base64,";
        }

        public String getKeyUrl(String type) {
            return "";
        }
    }
}
