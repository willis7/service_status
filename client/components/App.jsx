import React, {Component} from 'react';
import OutageSection from './status/OutageSection.jsx';
import OperationalSection from './status/OperationalSection.jsx';

class App extends Component{
    constructor(props){
        super(props);
        this.state = {
            operationals: [{id: 1, url: 'google.com'}],
            outages: [{id: 1, url: 'blah.com', time: '60'}]
        };
    }
    render(){
        return(
            <div>
            <OutageSection
                outages={this.state.outages}
            />
            <OperationalSection
                operationals={this.state.operationals}
            />
            </div>
        )
    }
}

export default App
