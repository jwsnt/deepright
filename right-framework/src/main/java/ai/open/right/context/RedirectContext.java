package ai.open.right.context;

import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.extern.slf4j.Slf4j;

public interface RedirectContext {

    public static final RedirectContext EMPTY = new EmptyContext();

    public static final Integer DEEPNESS = 1;

    // 递增
    public RedirectContext incrDeepness();

    public void setDeepness(Integer deepness);

    public Integer getDeepness();

    @JsonIgnore
    // 思考链（Workflow）起始的Query
    public String getOriginal();

    @JsonIgnore
    // 思考链（Workflow）上一次Query
    public String getPrevious();

    @JsonIgnore
    // 思考链（Workflow）当前初始Query
    public String getInitial();

    // 请求是否来自于FunCall的Merge
    @JsonIgnore
    public Boolean isFromFunMerge();

    // 请求是否来自于FunCall
    @JsonIgnore
    public Boolean isFromFunCall();

    @JsonIgnore
    public Boolean isEntry();

    @Slf4j
    public static class EmptyContext implements RedirectContext {

        @Override
        public RedirectContext incrDeepness() {
            this.warn();
            return this;
        }

        @Override
        public Integer getDeepness() {
            this.warn();
            return RedirectContext.DEEPNESS;
        }

        @Override
        public void setDeepness(Integer deepness) {
            this.warn();
            throw new IllegalStateException("Cannot set deepness on empty context");
        }

        @Override
        public String getOriginal() {
            this.warn();
            return null;
        }

        @Override
        public String getPrevious() {
            this.warn();
            return null;
        }

        @Override
        public String getInitial() {
            this.warn();
            return null;
        }

        @Override
        public Boolean isFromFunMerge() {
            this.warn();
            return false;
        }

        @Override
        public Boolean isFromFunCall() {
            this.warn();
            return false;
        }

        @Override
        public Boolean isEntry() {
            this.warn();
            return false;
        }

        public void warn() {
            if (log.isWarnEnabled()) {
                log.warn("The empty context can only be used in dev.");
            }
        }
    }
}
