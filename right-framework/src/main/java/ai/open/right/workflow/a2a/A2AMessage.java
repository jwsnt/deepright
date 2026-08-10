package ai.open.right.workflow.a2a;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.a2a.protocol.*;
import ai.open.right.workflow.flow.WorkflowTask;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.beanutils.PropertyUtils;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.Map;

@Getter
@Setter
public class A2AMessage {

    protected Message message;

    public A2AMessage(WorkflowTask workTask) throws Exception {
        Assert.hasText(workTask.getQuery(), "A2A message can not be empty");
        this.message =   workTask.getObjectQuery(MessageRequest.class).getMessage();
        Assert.notNull(this.message, "A2A Message parsing failed: " + workTask.getQuery());
    }

    // 支持嵌套属性访问，例如A.B.C
    public <T> T getMetadata(String key, Class<T> clazz) throws Exception {
        Object value = this.getMetadata(key);
        if (value == null) {
            return null;
        }
        if (value.getClass().isAssignableFrom(clazz)) {
            return clazz.cast(value);
        }
        return JsonUtils.transfer(value, clazz);
    }

    public Map<String, Object> getMetadata() throws Exception {
        return this.message.getMetadata();
    }

    public Object getMetadata(String key) throws Exception {
        if (CollectionUtils.isEmpty(this.message.getMetadata())) {
            return null;
        }
        return PropertyUtils.getNestedProperty(this.message.getMetadata(), key);
    }

    public <T> T getDataPart(int index, Class<T> clazz) throws Exception {
        Part part = this.getPart(index);
        return part != null ? JsonUtils.transfer(part.getData(), clazz) : null;
    }

    public <T> T getFirstDataPart(Class<T> clazz) throws Exception {
        Map<String, Object> part = this.getFirstDataPart();
        return part != null ? JsonUtils.transfer(part, clazz) : null;
    }

    public <T> T getLastDataPart(Class<T> clazz) throws Exception {
        Map<String, Object> part = this.getLastDataPart();
        return part != null ? JsonUtils.transfer(part, clazz) : null;
    }

    public Map<String, Object> getDataPart(int index) throws Exception {
        Part part = this.getPart(index);
        return part != null ? part.getData() : null;
    }

    public Map<String, Object> getFirstDataPart() throws Exception {
        Part part = this.getFirstPart(Part.DATA_KIND);
        return part != null ? part.getData() : null;
    }

    public Map<String, Object> getLastDataPart() throws Exception {
        Part part = this.getLastPart(Part.DATA_KIND);
        return part != null ? part.getData() : null;
    }

    public DataPart getObjectPart(int index) throws Exception {
        Map<String, Object> part = this.getDataPart(index);
        return part != null ? new DataPart(part) : null;
    }

    public DataPart getFirstObjectPart() throws Exception {
        Map<String, Object> part = this.getFirstDataPart();
        return part != null ? new DataPart(part) : null;
    }

    public DataPart getLastObjectPart() throws Exception {
        Map<String, Object> part = this.getLastDataPart();
        return part != null ? new DataPart(part) : null;
    }

    public FileData getFilePart(int index) throws Exception {
        Part part = this.getPart(index);
        return part != null ? part.getFile() : null;
    }

    public FileData getFirstFilePart() throws Exception {
        Part part = this.getFirstPart(Part.FILE_KIND);
        return part != null ? part.getFile() : null;
    }

    public FileData getLastFilePart() throws Exception {
        Part part = this.getLastPart(Part.FILE_KIND);
        return part != null ? part.getFile() : null;
    }

    public String getTextPart(int index) throws Exception {
        Part part = this.getPart(index);
        return part != null ? part.getText() : null;
    }

    public String getFirstTextPart() throws Exception {
        Part part = this.getFirstPart(Part.TEXT_KIND);
        return part != null ? part.getText() : null;
    }

    public String getLastTextPart() throws Exception {
        Part part = this.getLastPart(Part.TEXT_KIND);
        return part != null ? part.getText() : null;
    }

    public Part getFirstPart(String kind) throws Exception {
        if (CollectionUtils.isEmpty(this.message.getParts())) {
            return null;
        }
        for (Part part : this.message.getParts()) {
            if (part.isKind(kind)) {
                return part;
            }
        }
        return null;
    }

    public Part getLastPart(String kind) throws Exception {
        if (CollectionUtils.isEmpty(this.message.getParts())) {
            return null;
        }
        for (int index = this.message.getParts().size() - 1; index >= 0; index--) {
            Part part = this.message.getParts().get(index);
            if (part.isKind(kind)) {
                return part;
            }
        }
        return null;
    }

    public Part getPart(int index) throws Exception {
        if (CollectionUtils.isEmpty(this.message.getParts())) {
            return null;
        }
        Assert.isTrue(this.message.getParts().size() > index, "Part index exceeds the limit: " + index);
        Part part = this.message.getParts().get(index);
        if (!part.isKind(Part.TEXT_KIND)) {
            Assert.isTrue(this.message.getParts().size() > index, "Part's kind is not `text`': " + part.getKind());
        }
        return part;
    }

    public String getMessageId() {
        return this.message.getMessageId();
    }

    public String getContextId() {
        return this.message.getContextId();
    }

    public String getTaskId() {
        return this.message.getTaskId();
    }
}
