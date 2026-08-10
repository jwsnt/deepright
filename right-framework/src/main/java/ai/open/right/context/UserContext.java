package ai.open.right.context;

import ai.open.right.utils.JsonUtils;
import lombok.*;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

@Setter
@Getter
@ToString
@Builder
@AllArgsConstructor
@NoArgsConstructor
public class UserContext {

    public static final String UNKNOWN = "UNKNOWN";

    // 用户级别的Metadata, 多线程操作，不可调换，不会随传递丢失
    protected final Map<String, Object> metadata = new ConcurrentHashMap<String, Object>();

    protected String language;

    // 操作系统
    protected String system;

    // 设备
    protected String device;

    // 地区
    protected String region;

    // 手机品牌
    protected String brand;

    // 手机型号
    protected String model;

    // Token(非必需)
    protected String token;

    public void putMetadata(String key, Object value) {
        this.metadata.put(key, value);
    }

    public <T> T getMetadata(String key, Class<T> clazz) throws Exception {
        Assert.notNull(clazz, "The class can not be null");
        if (!MapUtils.isEmpty(this.metadata)) {
            Object val = this.metadata.get(key);
            if (val != null) {
                return clazz.isAssignableFrom(val.getClass()) ? clazz.cast(val) : JsonUtils.transfer(val, clazz);
            } else {
                return null;
            }
        } else {
            return null;
        }
    }

    public <T> T delMetadata(String key, Class<T> clazz) throws Exception {
        return !MapUtils.isEmpty(this.metadata) ? clazz.cast(this.metadata.remove(key)) : null;
    }

    public Object getMetadata(String key) {
        return MapUtils.getObject(this.metadata, key);
    }

    // 为入参UserContext填充默认属性，并改变原有UserContext
    public static UserContext setDefault(UserContext context) {
        context = context != null ? context : UserContext.builder().build();
        if (StringUtils.isEmpty(context.getLanguage())) {
            context.setLanguage(UserContext.UNKNOWN);
        }
        if (StringUtils.isEmpty(context.getSystem())) {
            context.setSystem(UserContext.UNKNOWN);
        }
        if (StringUtils.isEmpty(context.getDevice())) {
            context.setDevice(UserContext.UNKNOWN);
        }
        if (StringUtils.isEmpty(context.getRegion())) {
            context.setRegion(UserContext.UNKNOWN);
        }
        if (StringUtils.isEmpty(context.getBrand())) {
            context.setBrand(UserContext.UNKNOWN);
        }
        if (StringUtils.isEmpty(context.getModel())) {
            context.setModel(UserContext.UNKNOWN);
        }
        return context;
    }

    public static UserContext copyWithDevice(UserContext context, String device) {
        return UserContext.setDefault(UserContext.builder()
                .device(StringUtils.defaultString(device, context.getDevice()))
                .language(context.getLanguage())
                .region(context.getRegion())
                .brand(context.getBrand())
                .model(context.getModel())
                .build());
    }

    public static UserContext copy(UserContext context) {
        return UserContext.setDefault(UserContext.builder()
                .language(context.getLanguage())
                .device(context.getDevice())
                .region(context.getRegion())
                .brand(context.getBrand())
                .model(context.getModel())
                .build());
    }

    public static class UserContextChecker {

        public static void check(UserContext context) {
            Assert.notNull(context.getLanguage(), "Language can not be empty");
            Assert.notNull(context.getSystem(), "System can not be empty");
            Assert.notNull(context.getDevice(), "Device can not be empty");
            Assert.notNull(context.getRegion(), "Region can not be empty");
            Assert.notNull(context.getBrand(), "Brand can not be empty");
            Assert.notNull(context.getModel(), "Model can not be empty");
        }
    }
}
