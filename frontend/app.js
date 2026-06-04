const { useState, useEffect } = React;

function App() {
    const [tasks, setTasks] = useState([]);
    const [taskName, setTaskName] = useState('');
    const [statusMessage, setStatusMessage] = useState('');
    const [statusError, setStatusError] = useState(false);
    const [errorMessage, setErrorMessage] = useState('');
    const [loading, setLoading] = useState(true);

    const API_BASE_URL = window.API_BASE_URL || 'http://localhost:8080';

    useEffect(() => {
        loadTasks();
    }, []);

    async function checkHealth() {
        try {
            const response = await fetch(`${API_BASE_URL}/health`);
            const data = await response.json();
            setStatusMessage(`✅ Server Status: ${data.status}`);
            setStatusError(false);
        } catch (error) {
            setStatusMessage(`❌ Error: ${error.message}`);
            setStatusError(true);
        }
    }

    async function loadTasks() {
        setLoading(true);
        setErrorMessage('');

        try {
            const response = await fetch(`${API_BASE_URL}/tasks`);
            if (!response.ok) throw new Error('Failed to load tasks');
            const data = await response.json();
            setTasks(data.tasks || []);
        } catch (error) {
            setErrorMessage(`Error loading tasks: ${error.message}`);
            setTasks([]);
        } finally {
            setLoading(false);
        }
    }

    async function addTask() {
        setErrorMessage('');
        const trimmed = taskName.trim();
        if (!trimmed) {
            setErrorMessage('Please enter a task name');
            return;
        }

        try {
            const response = await fetch(`${API_BASE_URL}/tasks/add`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ task: trimmed }),
            });
            if (!response.ok) throw new Error('Failed to add task');
            setTaskName('');
            await loadTasks();
        } catch (error) {
            setErrorMessage(`Error adding task: ${error.message}`);
        }
    }

    function handleKeyPress(event) {
        if (event.key === 'Enter') {
            addTask();
        }
    }

    return React.createElement(
        'div',
        { className: 'container' },
        React.createElement('h1', null, '📋 Task Manager'),
        React.createElement(
            'div',
            { className: 'health-section' },
            React.createElement(
                'button',
                { className: 'health-button', onClick: checkHealth },
                'Check Server Health'
            ),
            React.createElement(
                'div',
                { className: `health-status ${statusMessage ? (statusError ? 'error active' : 'active') : ''}` },
                statusMessage
            )
        ),
        React.createElement(
            'div',
            { className: 'add-task-section' },
            React.createElement('h2', { style: { marginBottom: '15px', color: '#333', fontSize: '16px' } }, 'Add New Task'),
            React.createElement(
                'div',
                { className: `error-message ${errorMessage ? 'show' : ''}` },
                errorMessage
            ),
            React.createElement(
                'div',
                { className: 'form-group' },
                React.createElement('input', {
                    type: 'text',
                    id: 'taskInput',
                    value: taskName,
                    placeholder: 'Enter a new task...',
                    onChange: (event) => setTaskName(event.target.value),
                    onKeyPress: handleKeyPress,
                }),
                React.createElement(
                    'button',
                    { className: 'add-button', onClick: addTask },
                    'Add'
                )
            )
        ),
        React.createElement(
            'div',
            { className: 'tasks-section' },
            React.createElement(
                'div',
                { style: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '15px' } },
                React.createElement('h2', null, 'Tasks'),
                React.createElement(
                    'button',
                    { className: 'refresh-button', onClick: loadTasks },
                    'Refresh'
                )
            ),
            loading
                ? React.createElement('div', { className: 'loading' }, 'Loading tasks...')
                : tasks.length === 0
                    ? React.createElement('div', { className: 'empty-state' }, 'No tasks yet. Add one to get started!')
                    : React.createElement(
                        'ul',
                        { className: 'tasks-list' },
                        tasks.map((task, index) =>
                            React.createElement(
                                'li',
                                { key: index, className: 'task-item' },
                                React.createElement('span', { className: 'task-name' }, task),
                                React.createElement('span', { className: 'task-time' }, `Task ${index + 1}`)
                            )
                        )
                    )
        )
    );
}

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(React.createElement(App));
