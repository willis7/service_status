import React, {Component} from 'react';
import OutageSection from './status/OutageSection.jsx';

class App extends Component{
    constructor(props){
        super(props);
        this.state = {
            outages: [{id: 1, url: 'blah.com', time: '60'}]
        };
    }
    render(){
        return(
            <OutageSection
                outages={this.state.outages}
            />
        )
    }
}

export default App
