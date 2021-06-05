
import './App.css';
import React from 'react';
import ReactDOM from 'react-dom';

class App extends React.Component {


  componentDidMount(){
    console.log('mounted');
  }

  render() {
    return <h1>This is working</h1>;
  }
}

ReactDOM.render(
  <App />,
  document.getElementById('root')
);


export default App;
