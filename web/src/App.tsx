import { useState, useEffect } from 'react';
import './App.css';

function App() {
  const [isAlive, setIsAlive] = useState(false);

  useEffect(() => {
    fetch('/api/isalive')
      .then(response => response.json())
      .then(data => setIsAlive(data.is_alive));
  }, []);

  const turnOn = () => {
    fetch('/api/turnon').then(() => setIsAlive(true));
  };

  const turnOff = () => {
    fetch('/api/turnoff').then(() => setIsAlive(false));
  };

  return (
    <div className="App">
      <header className="App-header">
        <p>ESXi Manager</p>
        <div>
          <span>Server Status: </span>
          <span style={{ color: isAlive ? 'green' : 'red' }}>
            {isAlive ? 'Online' : 'Offline'}
          </span>
        </div>
        <button onClick={turnOn}>Turn On Server</button>
        <button onClick={turnOff}>Turn Off Server</button>
      </header>
    </div>
  );
}

export default App;
