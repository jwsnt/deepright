package ai.open.right;

import ai.open.right.protocol.ProtocolCode;
import com.fasterxml.jackson.core.JacksonException;
import lombok.Getter;
import lombok.extern.slf4j.Slf4j;

import java.io.IOException;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeoutException;

/**
 * @author shenjiawei
 */
@Getter
@Slf4j
public class WorkflowException extends RuntimeException {

    public static final String MAX_TOKEN = "MAX_TOKEN";

    protected Boolean exposure = false;

    protected Boolean silent = false;

    protected Boolean retry = false;

    protected Integer code = null;

    public WorkflowException() {
        this("", ProtocolCode.C500);
    }

    public WorkflowException(String e, Integer code) {
        super(e);
        this.code = code;
    }

    public WorkflowException(Throwable e, Integer code) {
        super(e.getCause() != null ? e.getCause() : e);
        this.code = code;
    }

    public WorkflowException(String e) {
        this(e, ProtocolCode.C500);
    }

    public WorkflowException(Throwable e) {
        this(e.getMessage());
        // 400 Bad Request
        if (IllegalArgumentException.class.isAssignableFrom(e.getClass())) {
            this.code = ProtocolCode.C400;
        }
    }

    public WorkflowException needExposure(Throwable e) {
        // 原始请求Silent为True时
        this.exposure = WorkflowException.exposure(e);
        return this;
    }

    public WorkflowException needSilent(Throwable e) {
        // 原始请求Silent为True时
        this.silent = WorkflowException.silent(e);
        return this;
    }

    public WorkflowException needExposure() {
        this.exposure = true;
        return this;
    }

    public WorkflowException needSilent() {
        this.silent = true;
        return this;
    }

    public WorkflowException dolog() {
        WorkflowException.dolog(this);
        return this;
    }

    public static boolean exposure(Throwable e) {
        Throwable t = e;
        while (t != null) {
            if (WorkflowException.class.isAssignableFrom(t.getClass())) {
                return WorkflowException.class.cast(t).getExposure();
            }
            t = t.getCause();
        }
        return false;
    }

    public static boolean silent(Throwable e) {
        Throwable t = e;
        while (t != null) {
            if (WorkflowException.class.isAssignableFrom(t.getClass())) {
                return WorkflowException.class.cast(t).getSilent();
            }
            t = t.getCause();
        }
        return false;
    }

    public static boolean retry(Throwable e) {
        if (!WorkflowException.class.equals(e.getClass())) {
            return false;
        }
        return WorkflowException.class.cast(e).getRetry();
    }

    public static int code(Throwable e, int def) {
        // Assert错误
        if (IllegalArgumentException.class.isAssignableFrom(e.getClass())) {
            if (log.isDebugEnabled()) {
                log.debug("Handle the IllegalArgumentException", e);
            }
            return ProtocolCode.C400;
        }
        // 多线程错误
        if (ExecutionException.class.isAssignableFrom(e.getClass())) {
            if (log.isDebugEnabled()) {
                log.debug("Handle the ExecutionException", e);
            }
            return ProtocolCode.C503;
        }
        // 多线程错误
        if (JacksonException.class.isAssignableFrom(e.getClass())) {
            if (log.isDebugEnabled()) {
                log.debug("Handle the JacksonException", e);
            }
            return ProtocolCode.C400;
        }
        // 多线程超时错误
        if (TimeoutException.class.isAssignableFrom(e.getClass())) {
            if (log.isDebugEnabled()) {
                log.debug("Handle the TimeoutException", e);
            }
            return ProtocolCode.C502;
        }
        // IO错误
        if (IOException.class.isAssignableFrom(e.getClass())) {
            if (log.isDebugEnabled()) {
                log.debug("Handle the IOException", e);
            }
            return ProtocolCode.C503;
        }
        // Workflow错误
        if (WorkflowException.class.isAssignableFrom(e.getClass())) {
            if (log.isDebugEnabled()) {
                log.debug("Handle the WorkflowException", e);
            }
            Integer code = WorkflowException.class.cast(e).code;
            return code != null ? code : ProtocolCode.C500;
        }
        if (log.isDebugEnabled()) {
            log.debug("Handle the Unknown Exception", e);
        }
        // 兜底非Workflow500
        return def;
    }

    public static int code(Throwable e) {
        return WorkflowException.code(e, ProtocolCode.C500);
    }

    public static WorkflowException create(String message, Integer code) {
        return new WorkflowException(message, code);
    }

    public static WorkflowException create(Throwable e, Integer code) {
        if (WorkflowException.class.isAssignableFrom(e.getClass())) {
            return WorkflowException.class.cast(e);
        } else {
            if (log.isErrorEnabled()) {
                log.error(e.getMessage(), e);
            }
            Throwable t = e.getCause();
            while (t != null && t.getCause() != null) {
                t = t.getCause();
            }
            return t != null ? new WorkflowException(t.getMessage(), code) : new WorkflowException(e.getMessage(), code);
        }
    }

    public static WorkflowException create(String message) {
        return new WorkflowException(message);
    }

    public static WorkflowException create(Throwable e) {
        return WorkflowException.create(e, WorkflowException.code(e));
    }

    public static void dolog(Throwable e, String message) {
        if (!WorkflowException.silent(e)) {
            log.error(message, e);
        } else if (log.isInfoEnabled()) {
            log.info(message, e);
        }
    }

    public static void dolog(Throwable e) {
        WorkflowException.dolog(e, e.getMessage());
    }


    public static void checkCondition(Boolean condition, Boolean exposure, Boolean silent, String message, Integer code) throws WorkflowException {
        if (condition) {
            WorkflowException exception = new WorkflowException(message, code);
            if (exposure) {
                exception.needExposure();
            }
            if (silent) {
                exception.needSilent();
            }
            throw exception;
        }
    }

    public static void checkCondition(Boolean condition, Boolean exposure, String message, Integer code) throws WorkflowException {
        WorkflowException.checkCondition(condition, exposure, false, message, code);
    }

    public static void checkCondition(Boolean condition, String message, Integer code) throws WorkflowException {
        WorkflowException.checkCondition(condition, false, false, message, code);
    }

    public static void checkCondition(Boolean condition, String message) throws WorkflowException {
        WorkflowException.checkCondition(condition, false, false, message, ProtocolCode.C500);
    }

    public static void checkSilent(Boolean condition, String message) throws WorkflowException {
        WorkflowException.checkCondition(condition, false, false, message, ProtocolCode.C915);
    }
}
