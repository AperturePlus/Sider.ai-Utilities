// Sider2API Chat UI
const messagesEl = document.getElementById('messages');
const statusEl = document.getElementById('status');
const inputEl = document.getElementById('input');
const sendBtn = document.getElementById('send');
const clearBtn = document.getElementById('clear');
const modelSelect = document.getElementById('model');
const thinkingCheckbox = document.getElementById('thinking');
const searchCheckbox = document.getElementById('search');

let conversationHistory = [];
let conversationId = null;  // 保存会话 ID
let parentMessageId = null; // 保存父消息 ID

// 自动调整 textarea 高度
inputEl.addEventListener('input', function() {
  this.style.height = 'auto';
  this.style.height = Math.min(this.scrollHeight, 120) + 'px';
});

// Enter 发送，Shift+Enter 换行
inputEl.addEventListener('keydown', function(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    sendMessage();
  }
});

sendBtn.addEventListener('click', sendMessage);
clearBtn.addEventListener('click', clearConversation);

function showStatus(message, type = '') {
  statusEl.textContent = message;
  statusEl.className = 'status show ' + type;
}

function hideStatus() {
  statusEl.className = 'status';
}

function addMessage(role, content, meta) {
  // 移除空状态提示
  const emptyState = messagesEl.querySelector('.empty-state');
  if (emptyState) {
    emptyState.remove();
  }

  const messageEl = document.createElement('div');
  messageEl.className = 'message ' + role;

  const label = document.createElement('div');
  label.className = 'message-label';
  label.textContent = role === 'user' ? '你' : 'AI';

  const bubble = document.createElement('div');
  bubble.className = 'message-bubble';
  bubble.textContent = content;

  messageEl.appendChild(label);
  messageEl.appendChild(bubble);

  if (meta && meta.usage) {
    const metaEl = document.createElement('div');
    metaEl.className = 'message-meta';
    metaEl.textContent = `输入: ${meta.usage.input_tokens || 0} tokens · 输出: ${meta.usage.output_tokens || 0} tokens`;
    messageEl.appendChild(metaEl);
  }

  messagesEl.appendChild(messageEl);
  messagesEl.scrollTop = messagesEl.scrollHeight;
  
  return messageEl;  // 返回消息元素，便于后续更新
}

// 添加或更新思考部分（在同一个消息气泡中）
function updateThinkingSection(assistantMsg, thinkingContent) {
  if (!assistantMsg) return;
  
  const bubble = assistantMsg.querySelector('.message-bubble');
  let thinkingSection = bubble.querySelector('.thinking-section');
  
  if (!thinkingSection) {
    thinkingSection = document.createElement('div');
    thinkingSection.className = 'thinking-section';
    bubble.insertBefore(thinkingSection, bubble.firstChild);
  }
  
  thinkingSection.innerHTML = `<div class="thinking-label">🧠 思考过程：</div><div class="thinking-content">${thinkingContent}</div>`;
  messagesEl.scrollTop = messagesEl.scrollHeight;
}

// 添加或更新回答部分
function updateAnswerSection(assistantMsg, answerContent) {
  if (!assistantMsg) return;
  
  const bubble = assistantMsg.querySelector('.message-bubble');
  let answerSection = bubble.querySelector('.answer-section');
  
  if (!answerSection) {
    answerSection = document.createElement('div');
    answerSection.className = 'answer-section';
    bubble.appendChild(answerSection);
  }
  
  answerSection.textContent = answerContent;
  messagesEl.scrollTop = messagesEl.scrollHeight;
}

// 添加搜索过程消息
function addSearchMessage(text) {
  let searchEl = messagesEl.querySelector('.message.search');
  
  if (!searchEl) {
    searchEl = document.createElement('div');
    searchEl.className = 'message search';
    
    const label = document.createElement('div');
    label.className = 'message-label';
    label.textContent = '🔍 搜索';
    
    const bubble = document.createElement('div');
    bubble.className = 'message-bubble search-bubble';
    
    searchEl.appendChild(label);
    searchEl.appendChild(bubble);
    messagesEl.appendChild(searchEl);
  }
  
  const bubble = searchEl.querySelector('.message-bubble');
  bubble.textContent = text;
  
  messagesEl.scrollTop = messagesEl.scrollHeight;
  return searchEl;
}

// 移除临时消息（search）
function removeTemporaryMessages() {
  const searchEl = messagesEl.querySelector('.message.search');
  if (searchEl) searchEl.remove();
}

function clearConversation() {
  conversationHistory = [];
  conversationId = null;
  parentMessageId = null;
  messagesEl.innerHTML = '<div class="empty-state">开始新的对话</div>';
  hideStatus();
}

async function sendMessage() {
  const message = inputEl.value.trim();
  if (!message || sendBtn.disabled) return;

  const model = modelSelect.value;
  const thinkEnabled = thinkingCheckbox.checked;
  const searchEnabled = searchCheckbox.checked;

  // 添加用户消息到界面
  addMessage('user', message);
  conversationHistory.push({ role: 'user', content: message });

  // 清空输入框
  inputEl.value = '';
  inputEl.style.height = 'auto';

  // 禁用发送按钮
  sendBtn.disabled = true;
  showStatus('正在发送...', '');

  // 用于累积流式响应
  let assistantMessageEl = null;
  let fullText = '';
  let thinkingText = '';
  let lastEventType = '';

  try {
    const payload = {
      model: model,
      messages: conversationHistory,
      stream: true,  // 启用流式响应
      metadata: {
        think_enabled: thinkEnabled,
        search_enabled: searchEnabled
      }
    };

    if (searchEnabled) {
      payload.tools = [{
        name: 'web_search',
        description: 'Search the web for information',
        input_schema: {
          type: 'object',
          properties: {
            query: { type: 'string', description: 'Search query' }
          },
          required: ['query']
        }
      }];
      payload.tool_choice = { type: 'auto' };
    }

    // 构建请求头，包含会话信息
    const headers = {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer ' + API_TOKEN
    };

    // 如果有会话 ID，添加到 URL 参数和请求头
    let url = '/v1/messages';
    if (conversationId) {
      url += '?cid=' + conversationId;
      headers['X-Conversation-ID'] = conversationId;
    }
    if (parentMessageId) {
      headers['X-Parent-Message-ID'] = parentMessageId;
    }

    const response = await fetch(url, {
      method: 'POST',
      headers: headers,
      body: JSON.stringify(payload)
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData?.error?.message || errorData?.error || '请求失败');
    }

    // 保存会话信息
    const newConversationId = response.headers.get('X-Conversation-ID');
    const newParentMessageId = response.headers.get('X-Assistant-Message-ID');
    
    if (newConversationId) {
      conversationId = newConversationId;
      console.log('会话 ID 已更新:', conversationId.substring(0, 12) + '...');
    }
    if (newParentMessageId) {
      parentMessageId = newParentMessageId;
      console.log('父消息 ID 已更新:', parentMessageId.substring(0, 12) + '...');
    }

    // 处理 SSE 流
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (!line.trim() || line.startsWith(':')) continue;

        if (line.startsWith('data:')) {
          const data = line.slice(5).trim();
          
          if (data === '[DONE]') {
            removeTemporaryMessages();
            continue;
          }

          try {
            const parsed = JSON.parse(data);
            console.log('📦 收到 SSE 事件:', parsed);
            
            // 后端返回的是 Anthropic 格式的 SSE（带自定义事件）
            const eventType = parsed.type;
            console.log('📝 事件类型:', eventType);
            
            // 处理消息开始
            if (eventType === 'message_start') {
              showStatus('🤔 准备中...', '');
            }
            
            // 处理推理开始（自定义事件）
            else if (eventType === 'reasoning_start') {
              console.log('🧠 推理开始');
              showStatus('🧠 思考中...', '');
              thinkingText = '';
              // 创建助手消息（如果还没有）
              if (!assistantMessageEl) {
                assistantMessageEl = addMessage('assistant', '');
              }
            }
            
            // 处理推理内容增量（自定义事件）
            else if (eventType === 'reasoning_delta') {
              const content = parsed.content || '';
              thinkingText += content;
              // 更新思考部分
              if (assistantMessageEl) {
                updateThinkingSection(assistantMessageEl, thinkingText);
              }
              console.log('🧠 推理内容:', content.substring(0, 50) + '...');
            }
            
            // 处理推理结束（自定义事件）
            else if (eventType === 'reasoning_end') {
              console.log('🧠 推理结束');
              showStatus('💡 思考完成', 'success');
            }
            
            // 处理搜索开始（自定义事件）
            else if (eventType === 'search_start') {
              const toolName = parsed.tool_name;
              console.log('🔍 搜索开始:', toolName);
              showStatus('🔍 正在搜索...', '');
              addSearchMessage('正在搜索相关信息...');
            }
            
            // 处理搜索结果（自定义事件）
            else if (eventType === 'search_result') {
              const result = parsed.result;
              console.log('📚 搜索结果:', result);
              
              // 格式化搜索结果
              let searchText = '🔍 找到相关信息：\n\n';
              
              // 检查是否是 search 类型的结果
              if (result.search && result.search.search_snippets) {
                const snippets = result.search.search_snippets;
                Object.keys(snippets).slice(0, 3).forEach(key => {
                  const snippet = snippets[key];
                  searchText += `📌 ${snippet.title}\n`;
                  searchText += `${snippet.snippet}\n`;
                  searchText += `🔗 ${snippet.link}\n\n`;
                });
              } else {
                searchText += JSON.stringify(result, null, 2);
              }
              
              // 更新搜索消息
              const searchEl = messagesEl.querySelector('.message.search:last-child');
              if (searchEl) {
                const bubble = searchEl.querySelector('.message-bubble');
                bubble.textContent = searchText;
              }
            }
            
            // 处理内容块开始
            else if (eventType === 'content_block_start') {
              if (!assistantMessageEl) {
                assistantMessageEl = addMessage('assistant', '');
              }
              // 移除搜索消息
              removeTemporaryMessages();
              showStatus('💬 回复中...', '');
            }
            
            // 处理内容块增量（文本内容）
            else if (eventType === 'content_block_delta') {
              const deltaText = parsed.delta?.text || '';
              fullText += deltaText;
              
              if (assistantMessageEl) {
                // 更新回答部分
                updateAnswerSection(assistantMessageEl, fullText);
                messagesEl.scrollTop = messagesEl.scrollHeight;
              }
            }
            
            // 处理内容块结束
            else if (eventType === 'content_block_stop') {
              // 内容块结束，但可能还有其他块
            }
            
            // 处理消息增量（最终状态）
            else if (eventType === 'message_delta') {
              const stopReason = parsed.delta?.stop_reason;
              if (stopReason === 'end_turn') {
                showStatus('✅ 完成', 'success');
                removeTemporaryMessages();
              }
            }
            
            // 处理消息结束
            else if (eventType === 'message_stop') {
              // 流结束
              hideStatus();
            }

          } catch (e) {
            console.error('解析 SSE 事件失败:', e, data);
          }
        }
      }
    }

    // 流结束，保存完整消息到历史
    const assistantMessage = fullText || '[空响应]';
    conversationHistory.push({ role: 'assistant', content: assistantMessage });

    // 如果没有创建消息元素（说明没有文本内容），创建一个
    if (!assistantMessageEl && assistantMessage) {
      addMessage('assistant', assistantMessage);
    }

    hideStatus();

  } catch (error) {
    console.error('Error:', error);
    showStatus(error.message || '请求失败', 'error');
    
    // 移除失败的用户消息
    conversationHistory.pop();
    const lastMessage = messagesEl.lastElementChild;
    if (lastMessage && lastMessage.classList.contains('user')) {
      lastMessage.remove();
    }
    
    removeTemporaryMessages();
  } finally {
    sendBtn.disabled = false;
    inputEl.focus();
  }
}

// 页面加载完成后聚焦输入框
window.addEventListener('load', () => {
  inputEl.focus();
});
